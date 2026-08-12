/**
 * LoadingSpinner - themed spinner component
 * Uses CSS theme variables so it renders correctly in both light and dark mode.
 */

import React from 'react';

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg' | 'xl';
  label?: string;
  fullScreen?: boolean;
}

const sizeMap: Record<NonNullable<LoadingSpinnerProps['size']>, string> = {
  sm: 'w-4 h-4 border-2',
  md: 'w-8 h-8 border-[3px]',
  lg: 'w-12 h-12 border-4',
  xl: 'w-16 h-16 border-4',
};

const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  size = 'md',
  label,
  fullScreen = false,
}) => {
  const spinner = (
    <div
      className={`${sizeMap[size]} rounded-full animate-spin`}
      style={{
        borderColor: 'var(--color-border)',
        borderTopColor: 'var(--color-primary)',
      }}
      role="status"
      aria-label={label || 'Loading'}
    />
  );

  if (fullScreen) {
    return (
      <div className="flex flex-col items-center justify-center min-h-screen gap-4">
        {spinner}
        {label && (
          <p className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
            {label}
          </p>
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center justify-center gap-2">
      {spinner}
      {label && (
        <p className="text-sm" style={{ color: 'var(--color-text-secondary)' }}>
          {label}
        </p>
      )}
    </div>
  );
};

export default LoadingSpinner;
