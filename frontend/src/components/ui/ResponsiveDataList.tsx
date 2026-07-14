import type { ReactNode } from 'react';

export interface ResponsiveDataListProps {
  desktop: ReactNode;
  mobile: ReactNode;
  desktopLabel: string;
  mobileLabel: string;
  className?: string;
}

export default function ResponsiveDataList({
  desktop,
  mobile,
  desktopLabel,
  mobileLabel,
  className,
}: ResponsiveDataListProps) {
  const classes = ['ui-responsive-data-list', className ?? ''].filter(Boolean).join(' ');

  return (
    <div className={classes}>
      <section className="ui-responsive-data-list__desktop" aria-label={desktopLabel}>
        {desktop}
      </section>
      <section className="ui-responsive-data-list__mobile" aria-label={mobileLabel}>
        {mobile}
      </section>
    </div>
  );
}
