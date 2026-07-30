'use client';

import React from 'react';
import { ThemeProvider, useTheme } from '../components/ThemeProvider';
import { ThemeToggle } from '../components/ThemeToggle';

// ============================================================================
// Layout with Theme Support
// ============================================================================

interface LayoutProps {
  children: React.ReactNode;
  title?: string;
}

export function WalletLayout({ children, title = 'TigerWallet' }: LayoutProps) {
  const { theme, isDark } = useTheme();
  
  return (
    <div className={`min-h-screen ${isDark ? 'bg-gray-900' : 'bg-gray-50'}`}>
      {/* Header with Theme Toggle */}
      <header className={`sticky top-0 z-50 ${isDark ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200'} border-b`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <h1 className={`text-xl font-bold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                {title}
              </h1>
            </div>
            <div className="flex items-center space-x-4">
              <ThemeToggle />
            </div>
          </div>
        </div>
      </header>
      
      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {children}
      </main>
      
      {/* Footer */}
      <footer className={`${isDark ? 'bg-gray-800 border-gray-700' : 'bg-white border-gray-200'} border-t mt-auto`}>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <p className={`text-center text-sm ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
            © 2026 TigerWallet. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}

// ============================================================================
// Card Component with Theme Support
// ============================================================================

interface CardProps {
  children: React.ReactNode;
  className?: string;
  title?: string;
}

export function ThemedCard({ children, className = '', title }: CardProps) {
  const { isDark } = useTheme();
  
  return (
    <div className={`rounded-lg shadow-md p-6 ${isDark ? 'bg-gray-800' : 'bg-white'} ${className}`}>
      {title && (
        <h3 className={`text-lg font-semibold mb-4 ${isDark ? 'text-white' : 'text-gray-900'}`}>
          {title}
        </h3>
      )}
      {children}
    </div>
  );
}

// ============================================================================
// Button Component with Theme Support
// ============================================================================

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger';
  size?: 'sm' | 'md' | 'lg';
}

export function ThemedButton({ 
  children, 
  variant = 'primary', 
  size = 'md',
  className = '',
  ...props 
}: ButtonProps) {
  const { isDark } = useTheme();
  
  const baseStyles = 'rounded-lg font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2';
  
  const variantStyles = {
    primary: isDark 
      ? 'bg-blue-600 hover:bg-blue-700 text-white focus:ring-blue-500' 
      : 'bg-blue-600 hover:bg-blue-700 text-white focus:ring-blue-500',
    secondary: isDark
      ? 'bg-gray-700 hover:bg-gray-600 text-white focus:ring-gray-500'
      : 'bg-gray-200 hover:bg-gray-300 text-gray-900 focus:ring-gray-500',
    danger: 'bg-red-600 hover:bg-red-700 text-white focus:ring-red-500',
  };
  
  const sizeStyles = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg',
  };
  
  return (
    <button 
      className={`${baseStyles} ${variantStyles[variant]} ${sizeStyles[size]} ${className}`}
      {...props}
    >
      {children}
    </button>
  );
}

// ============================================================================
// Input Component with Theme Support
// ============================================================================

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

export function ThemedInput({ label, error, className = '', ...props }: InputProps) {
  const { isDark } = useTheme();
  
  return (
    <div className="mb-4">
      {label && (
        <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
          {label}
        </label>
      )}
      <input
        className={`w-full px-3 py-2 rounded-lg border ${
          error 
            ? 'border-red-500 focus:ring-red-500' 
            : isDark 
              ? 'border-gray-600 bg-gray-700 text-white focus:ring-blue-500' 
              : 'border-gray-300 bg-white text-gray-900 focus:ring-blue-500'
        } focus:outline-none focus:ring-2 focus:ring-offset-0 transition-colors`}
        {...props}
      />
      {error && (
        <p className="mt-1 text-sm text-red-500">{error}</p>
      )}
    </div>
  );
}

// ============================================================================
// Select Component with Theme Support
// ============================================================================

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  options: { value: string; label: string }[];
}

export function ThemedSelect({ label, options, className = '', ...props }: SelectProps) {
  const { isDark } = useTheme();
  
  return (
    <div className="mb-4">
      {label && (
        <label className={`block text-sm font-medium mb-1 ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
          {label}
        </label>
      )}
      <select
        className={`w-full px-3 py-2 rounded-lg border ${
          isDark 
            ? 'border-gray-600 bg-gray-700 text-white' 
            : 'border-gray-300 bg-white text-gray-900'
        } focus:outline-none focus:ring-2 focus:ring-blue-500`}
        {...props}
      >
        {options.map(option => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}

// ============================================================================
// Badge Component with Theme Support
// ============================================================================

interface BadgeProps {
  children: React.ReactNode;
  variant?: 'success' | 'warning' | 'danger' | 'info';
}

export function ThemedBadge({ children, variant = 'info' }: BadgeProps) {
  const variantStyles = {
    success: 'bg-green-100 text-green-800',
    warning: 'bg-yellow-100 text-yellow-800',
    danger: 'bg-red-100 text-red-800',
    info: 'bg-blue-100 text-blue-800',
  };
  
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${variantStyles[variant]}`}>
      {children}
    </span>
  );
}

// ============================================================================
// Modal Component with Theme Support
// ============================================================================

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}

export function ThemedModal({ isOpen, onClose, title, children }: ModalProps) {
  const { isDark } = useTheme();
  
  if (!isOpen) return null;
  
  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen px-4">
        {/* Backdrop */}
        <div 
          className="fixed inset-0 bg-black bg-opacity-50"
          onClick={onClose}
        />
        
        {/* Modal Content */}
        <div className={`relative rounded-lg shadow-xl p-6 w-full max-w-md ${
          isDark ? 'bg-gray-800' : 'bg-white'
        }`}>
          {title && (
            <div className="flex justify-between items-center mb-4">
              <h3 className={`text-lg font-semibold ${isDark ? 'text-white' : 'text-gray-900'}`}>
                {title}
              </h3>
              <button
                onClick={onClose}
                className={`${isDark ? 'text-gray-400 hover:text-gray-300' : 'text-gray-400 hover:text-gray-500'}`}
              >
                ✕
              </button>
            </div>
          )}
          {children}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// Table Component with Theme Support
// ============================================================================

interface TableProps {
  children: React.ReactNode;
}

export function ThemedTable({ children }: TableProps) {
  const { isDark } = useTheme();
  
  return (
    <div className={`overflow-x-auto rounded-lg border ${isDark ? 'border-gray-700' : 'border-gray-200'}`}>
      <table className={`min-w-full divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
        {children}
      </table>
    </div>
  );
}

export function ThemedTableHead({ children }: { children: React.ReactNode }) {
  const { isDark } = useTheme();
  
  return (
    <thead className={isDark ? 'bg-gray-800' : 'bg-gray-50'}>
      {children}
    </thead>
  );
}

export function ThemedTableBody({ children }: { children: React.ReactNode }) {
  const { isDark } = useTheme();
  
  return (
    <tbody className={`${isDark ? 'bg-gray-900' : 'bg-white'} divide-y ${isDark ? 'divide-gray-700' : 'divide-gray-200'}`}>
      {children}
    </tbody>
  );
}

export function ThemedTableRow({ children }: { children: React.ReactNode }) {
  return <tr>{children}</tr>;
}

export function ThemedTableCell({ children }: { children: React.ReactNode }) {
  const { isDark } = useTheme();
  
  return (
    <td className={`px-6 py-4 whitespace-nowrap text-sm ${isDark ? 'text-gray-300' : 'text-gray-700'}`}>
      {children}
    </td>
  );
}

// ============================================================================
// Export ThemeProvider and useTheme for external use
// ============================================================================

export { ThemeProvider, useTheme };
