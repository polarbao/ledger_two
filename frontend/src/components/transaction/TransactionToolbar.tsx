import type { FormEvent } from 'react';
import { Download, Filter, ListChecks, Search } from 'lucide-react';
import Button from '../ui/Button';
import SegmentedControl from '../ui/SegmentedControl';
import type { TransactionQuickType } from './transactionsPageModel';

interface TransactionToolbarProps {
  month: string;
  keyword: string;
  type: TransactionQuickType;
  activeFilterCount: number;
  batchMode: boolean;
  canWrite: boolean;
  canExport: boolean;
  onMonthChange: (month: string) => void;
  onKeywordChange: (keyword: string) => void;
  onTypeChange: (type: TransactionQuickType) => void;
  onOpenFilters: () => void;
  onToggleBatch: () => void;
  onExport: () => void;
}

const typeOptions = [
  { value: '', label: '全部' },
  { value: 'expense', label: '支出' },
  { value: 'income', label: '收入' },
  { value: 'shared_expense', label: '共同' },
  { value: 'settlement', label: '结算' },
] as const;

export default function TransactionToolbar({
  month,
  keyword,
  type,
  activeFilterCount,
  batchMode,
  canWrite,
  canExport,
  onMonthChange,
  onKeywordChange,
  onTypeChange,
  onOpenFilters,
  onToggleBatch,
  onExport,
}: TransactionToolbarProps) {
  const handleSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    onKeywordChange(String(data.get('keyword') || '').trim());
  };

  return (
    <div className="transactions-toolbar">
      <div className="transactions-toolbar__primary">
        <label className="transactions-toolbar__month">
          <span className="sr-only">账期月份</span>
          <input type="month" value={month} onChange={(event) => onMonthChange(event.target.value)} />
        </label>
        <form key={keyword} className="transactions-toolbar__search" onSubmit={handleSearch}>
          <Search size={18} aria-hidden="true" />
          <input name="keyword" type="search" defaultValue={keyword} placeholder="搜索标题或备注" aria-label="搜索流水" />
          <button type="submit">搜索</button>
        </form>
        <Button variant="secondary" startIcon={<Filter size={17} />} onClick={onOpenFilters}>
          更多筛选{activeFilterCount > 0 ? ` ${activeFilterCount}` : ''}
        </Button>
      </div>
      <div className="transactions-toolbar__secondary">
        <SegmentedControl
          ariaLabel="账单类型"
          value={type}
          options={typeOptions}
          onChange={onTypeChange}
        />
        <div className="transactions-toolbar__actions">
          {canExport ? (
            <Button variant="secondary" startIcon={<Download size={17} />} onClick={onExport}>导出</Button>
          ) : null}
          {canWrite ? (
            <Button
              variant={batchMode ? 'primary' : 'secondary'}
              startIcon={<ListChecks size={17} />}
              onClick={onToggleBatch}
            >
              {batchMode ? '退出批量' : '批量管理'}
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );
}
