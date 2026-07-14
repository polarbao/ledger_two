import { X } from 'lucide-react';
import Button from './Button';

export interface ActiveFilterChip {
  key: string;
  label: string;
}

export interface ActiveFilterChipsProps {
  filters: ActiveFilterChip[];
  onRemove: (key: string) => void;
  onClear: () => void;
}

export default function ActiveFilterChips({ filters, onRemove, onClear }: ActiveFilterChipsProps) {
  if (filters.length === 0) return null;

  return (
    <div className="ui-active-filters" aria-label="当前筛选条件">
      <span className="ui-active-filters__label">已筛选</span>
      <div className="ui-active-filters__list">
        {filters.map((filter) => (
          <button
            key={filter.key}
            type="button"
            className="ui-active-filter-chip"
            aria-label={`移除筛选：${filter.label}`}
            title={`移除筛选：${filter.label}`}
            onClick={() => onRemove(filter.key)}
          >
            <span>{filter.label}</span>
            <X size={14} aria-hidden="true" />
          </button>
        ))}
      </div>
      <Button variant="ghost" onClick={onClear}>清除全部</Button>
    </div>
  );
}
