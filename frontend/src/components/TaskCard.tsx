import React from 'react';
import type { Task } from '../types';

interface TaskCardProps {
  task: Task;
  onClick: () => void;
}

const TYPE_COLORS: Record<string, { bg: string; color: string }> = {
  feature: { bg: '#dbeafe', color: '#1d4ed8' },
  fix: { bg: '#fee2e2', color: '#b91c1c' },
  refactor: { bg: '#fef3c7', color: '#92400e' },
  chore: { bg: '#f3e8ff', color: '#6b21a8' },
};

const STATUS_COLORS: Record<string, { bg: string; color: string }> = {
  pending: { bg: '#f3f4f6', color: '#6b7280' },
  planning: { bg: '#dbeafe', color: '#2563eb' },
  plan_review: { bg: '#e0e7ff', color: '#4338ca' },
  developing: { bg: '#fef3c7', color: '#d97706' },
  reviewing: { bg: '#ede9fe', color: '#7c3aed' },
  merging: { bg: '#cffafe', color: '#0891b2' },
  completed: { bg: '#dcfce7', color: '#16a34a' },
  failed: { bg: '#fee2e2', color: '#dc2626' },
};

const STATUS_LABELS: Record<string, string> = {
  pending: 'Pending',
  planning: 'Planning',
  plan_review: 'Plan Review',
  developing: 'Developing',
  reviewing: 'Code Review',
  merging: 'Merging',
  completed: 'Completed',
  failed: 'Failed',
};

function formatTimestamp(ts: string): string {
  const date = new Date(ts);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength).trimEnd() + '...';
}

const TaskCard: React.FC<TaskCardProps> = ({ task, onClick }) => {
  const [hovered, setHovered] = React.useState(false);

  const typeStyle = TYPE_COLORS[task.type] || { bg: '#f3f4f6', color: '#6b7280' };
  const statusStyle = STATUS_COLORS[task.status] || { bg: '#f3f4f6', color: '#6b7280' };
  const statusLabel = STATUS_LABELS[task.status] || task.status;
  const displayTitle = task.title || truncate(task.description, 80);

  return (
    <div
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        padding: '16px 20px',
        backgroundColor: hovered ? '#f8fafc' : '#ffffff',
        border: '1px solid',
        borderColor: hovered ? '#3b82f6' : '#e5e7eb',
        borderRadius: 8,
        cursor: 'pointer',
        transition: 'all 0.2s ease',
        boxShadow: hovered
          ? '0 4px 12px rgba(59, 130, 246, 0.12)'
          : '0 1px 3px rgba(0, 0, 0, 0.06)',
        transform: hovered ? 'translateY(-1px)' : 'translateY(0)',
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
      }}
    >
      <div
        style={{
          fontSize: 15,
          fontWeight: 600,
          color: '#1f2937',
          lineHeight: 1.4,
        }}
      >
        {displayTitle}
      </div>

      <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <span
          style={{
            display: 'inline-block',
            padding: '2px 10px',
            borderRadius: 9999,
            fontSize: 12,
            fontWeight: 600,
            backgroundColor: typeStyle.bg,
            color: typeStyle.color,
          }}
        >
          {task.type}
        </span>
        <span
          style={{
            display: 'inline-block',
            padding: '2px 10px',
            borderRadius: 9999,
            fontSize: 12,
            fontWeight: 600,
            backgroundColor: statusStyle.bg,
            color: statusStyle.color,
          }}
        >
          {statusLabel}
        </span>
      </div>

      <div
        style={{
          display: 'flex',
          gap: 16,
          fontSize: 12,
          color: '#9ca3af',
        }}
      >
        <span>Created: {formatTimestamp(task.created_at)}</span>
        {task.updated_at && task.updated_at !== task.created_at && (
          <span>Updated: {formatTimestamp(task.updated_at)}</span>
        )}
      </div>
    </div>
  );
};

export default TaskCard;
