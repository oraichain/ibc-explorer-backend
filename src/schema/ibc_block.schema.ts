import * as mongoose from 'mongoose';
import { IbcBlockType } from '../types/schemaTypes/ibc_block.interface';
export const IbcBlockSchema = new mongoose.Schema(
  {
    height: Number,
    hash: String,
    txn: Number,
    time: Number,
    proposer: String,
  },
  { versionKey: false },
);
IbcBlockSchema.index({ hash: -1 }, { unique: true });
IbcBlockSchema.index({ height: -1 }, { background: true });

IbcBlockSchema.statics = {
  async findLatestBlock(): Promise<IbcBlockType> {
    return this.findOne()
      .sort({ height: -1 })
  },
};
