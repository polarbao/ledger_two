import { api } from './client';
import type {
  BalanceResponse,
  SettlementResponse,
  CreateSettlementPayload,
} from '../types/settlement';

/**
 * @brief 结算中心 API 端点连接器
 */
export const settlementApi = {
  /**
   * @brief 获取当前两端未结清轧差数据
   * @return Promise<BalanceResponse> 轧差统计响应体
   */
  getBalance: () =>
    api.get<BalanceResponse>('/api/settlements/balance'),

  /**
   * @brief 获取结算明细历史列表
   * @param month 可选月份过滤参数 (如 "2026-06")
   * @return Promise<SettlementResponse[]> 结算流水数组
   */
  getSettlements: (month?: string) => {
    const query = month ? `?month=${encodeURIComponent(month)}` : '';
    return api.get<SettlementResponse[]>(`/api/settlements${query}`);
  },

  /**
   * @brief 提交一笔结算补款清偿动作
   * @param payload 结算创建请求载荷
   * @return Promise<SettlementResponse> 返回生成的结算明细结果
   */
  createSettlement: (payload: CreateSettlementPayload) =>
    api.post<SettlementResponse>('/api/settlements', payload),
};
