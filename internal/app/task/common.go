package task

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/constant"
	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model"
	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/dto"
	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/entity"
	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/model/vo"
	"github.com/bianjieai/iobscan-ibc-explorer-backend/internal/app/utils"
	"github.com/sirupsen/logrus"
)

type segment struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
}

func getHistorySegment() ([]*segment, error) {
	first, err := ibcTxRepo.FirstHistory()
	if err != nil {
		return nil, err
	}

	latest, err := ibcTxRepo.LatestHistory()
	if err != nil {
		return nil, err
	}

	start := time.Unix(first.CreateAt, 0)
	startUnix := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local).Unix()
	end := time.Unix(latest.CreateAt, 0)
	endUnix := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 59, time.Local).Unix()

	var step int64 = 12 * 3600
	var segments []*segment
	for temp := startUnix; temp < endUnix; temp += step {
		segments = append(segments, &segment{
			StartTime: temp,
			EndTime:   temp + step - 1,
		})
	}
	return segments, nil
}

func getSegment() ([]*segment, error) {
	first, err := ibcTxRepo.First()
	if err != nil {
		return nil, err
	}

	start := time.Unix(first.CreateAt, 0)
	startUnix := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local).Unix()
	end := time.Now()
	endUnix := time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 59, time.Local).Unix()

	var step int64 = 24 * 3600
	var segments []*segment
	for temp := startUnix; temp < endUnix; temp += step {
		segments = append(segments, &segment{
			StartTime: temp,
			EndTime:   temp + step - 1,
		})
	}

	return segments, nil
}

// todayUnix 获取今日第一秒和最后一秒的时间戳
func todayUnix() (int64, int64) {
	now := time.Now()
	startUnix := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).Unix()
	endUnix := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 59, time.Local).Unix()
	return startUnix, endUnix
}

// yesterdayUnix 获取昨日第一秒和最后一秒的时间戳
func yesterdayUnix() (int64, int64) {
	date := time.Now().AddDate(0, 0, -1)
	startUnix := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local).Unix()
	endUnix := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 59, time.Local).Unix()
	return startUnix, endUnix
}

func isConnectionErr(err error) bool {
	return strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "i/o timeout") ||
		strings.Contains(err.Error(), "unsupported protocol scheme")
}

func getAllChainMap() (map[string]*entity.ChainConfig, error) {
	allChainList, err := chainConfigRepo.FindAll()
	if err != nil {
		return nil, err
	}

	allChainMap := make(map[string]*entity.ChainConfig)
	for _, v := range allChainList {
		allChainMap[v.ChainId] = v
	}

	return allChainMap, err
}

func matchDcInfo(scChainId, scPort, scChannel string, allChainMap map[string]*entity.ChainConfig) (dcChainId, dcPort, dcChannel string) {
	if allChainMap == nil || allChainMap[scChainId] == nil {
		return
	}

	for _, ibcInfo := range allChainMap[scChainId].IbcInfo {
		for _, path := range ibcInfo.Paths {
			if path.PortId == scPort && path.ChannelId == scChannel {
				dcChainId = path.ChainId
				dcPort = path.Counterparty.PortId
				dcChannel = path.Counterparty.ChannelId
				return
			}
		}
	}

	return
}

// getRootDenom get root denom by denom path
//   - fullPath full fullPath, eg："transfer/channel-1/uiris", "uatom"
func getRootDenom(fullPath string) string {
	split := strings.Split(fullPath, "/")
	return split[len(split)-1]
}

// splitFullPath get denom path and root denom from denom path
//   - fullPath full fullPath, eg："transfer/channel-1/uiris", "uatom"
func splitFullPath(fullPath string) (denomPath, rootDenom string) {
	pathSplits := strings.Split(fullPath, "/")
	denomPath = strings.Join(pathSplits[0:len(pathSplits)-1], "/")
	rootDenom = pathSplits[len(pathSplits)-1]
	return
}

// calculateIbcHash calculate denom hash by denom path
//   - fullPath full fullPath, eg："transfer/channel-1/uiris", "uatom"
func calculateIbcHash(fullPath string) string {
	if len(strings.Split(fullPath, "/")) == 1 {
		return fullPath
	}

	hash := utils.Sha256(fullPath)
	return fmt.Sprintf("%s/%s", constant.IBCTokenPreFix, strings.ToUpper(hash))
}

// traceDenom trace denom path, parse denom info
//   - fullDenomPath denom full path，eg："transfer/channel-1/uiris", "uatom"
func traceDenom(fullDenomPath, chainId string, allChainMap map[string]*entity.ChainConfig) *entity.IBCDenom {
	unix := time.Now().Unix()
	denom := calculateIbcHash(fullDenomPath)
	rootDenom := getRootDenom(fullDenomPath)
	if !strings.HasPrefix(denom, constant.IBCTokenPreFix) { // base denom
		return &entity.IBCDenom{
			ChainId:          chainId,
			Denom:            denom,
			PrevDenom:        "",
			PrevChainId:      "",
			BaseDenom:        denom,
			BaseDenomChainId: chainId,
			DenomPath:        "",
			RootDenom:        rootDenom,
			IsBaseDenom:      true,
			CreateAt:         unix,
			UpdateAt:         unix,
		}
	}

	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("trace denom: %s, chain: %s, full path: %s, error. %v ", denom, chainId, fullDenomPath, err)
		}
	}()

	var currentChainId string
	var isBaseDenom bool
	currentChainId = chainId
	pathSplits := strings.Split(fullDenomPath, "/")
	denomPath := strings.Join(pathSplits[0:len(pathSplits)-1], "/")
	var TraceDenomList []*dto.DenomSimpleDTO
	TraceDenomList = append(TraceDenomList, &dto.DenomSimpleDTO{
		Denom:   denom,
		ChainId: chainId,
	})

	for {
		if len(pathSplits) <= 1 {
			break
		}

		currentPort, currentChannel := pathSplits[0], pathSplits[1]
		tempPrevChainId, tempPrevPort, tempPrevChannel := matchDcInfo(currentChainId, currentPort, currentChannel, allChainMap)
		if tempPrevChainId == "" { // trace to end
			break
		} else {
			TraceDenomList = append(TraceDenomList, &dto.DenomSimpleDTO{
				Denom:   calculateIbcHash(strings.Join(pathSplits[2:], "/")),
				ChainId: tempPrevChainId,
			})
		}

		currentChainId, currentPort, currentChannel = tempPrevChainId, tempPrevPort, tempPrevChannel
		pathSplits = pathSplits[2:]
	}

	var prevDenom, prevChainId, baseDenom, baseDenomChainId string
	if len(TraceDenomList) == 1 { // denom is base denom
		isBaseDenom = true
		baseDenom = denom
		baseDenomChainId = chainId
	} else {
		isBaseDenom = false
		prevDenom = TraceDenomList[1].Denom
		prevChainId = TraceDenomList[1].ChainId
		baseDenom = TraceDenomList[len(TraceDenomList)-1].Denom
		baseDenomChainId = TraceDenomList[len(TraceDenomList)-1].ChainId
	}

	return &entity.IBCDenom{
		ChainId:          chainId,
		Denom:            denom,
		PrevDenom:        prevDenom,
		PrevChainId:      prevChainId,
		BaseDenom:        baseDenom,
		BaseDenomChainId: baseDenomChainId,
		DenomPath:        denomPath,
		RootDenom:        rootDenom,
		IsBaseDenom:      isBaseDenom,
		CreateAt:         unix,
		UpdateAt:         unix,
	}
}

// calculateNextDenomPath calculate full denom path of next hop.
// return full denom path and cross back identification
func calculateNextDenomPath(packet model.Packet) (string, bool) {
	prefixSc := fmt.Sprintf("%s/%s/", packet.SourcePort, packet.SourceChannel)
	prefixDc := fmt.Sprintf("%s/%s/", packet.DestinationPort, packet.DestinationChannel)
	denomPath := packet.Data.Denom
	if strings.HasPrefix(denomPath, prefixSc) { // transfer to prev chain
		denomPath = strings.Replace(denomPath, prefixSc, "", 1)
		return denomPath, true
	} else {
		denomPath = fmt.Sprintf("%s%s", prefixDc, denomPath)
		return denomPath, false
	}
}

// queryClientState 查询lcd client_state_path接口
func queryClientState(lcd, apiPath, port, channel string) (*vo.ClientStateResp, error) {
	apiPath = strings.ReplaceAll(apiPath, replaceHolderChannel, channel)
	apiPath = strings.ReplaceAll(apiPath, replaceHolderPort, port)
	url := fmt.Sprintf("%s%s", lcd, apiPath)
	bz, err := utils.HttpGet(url)
	if err != nil {
		return nil, err
	}

	var resp vo.ClientStateResp
	err = json.Unmarshal(bz, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}