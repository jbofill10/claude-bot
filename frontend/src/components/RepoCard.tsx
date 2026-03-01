import React from 'react';
import type { Repo } from '../types';

interface RepoCardProps {
  repo: Repo;
  onClick: () => void;
}

const RepoCard: React.FC<RepoCardProps> = ({ repo, onClick }) => {
  const [hovered, setHovered] = React.useState(false);

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
      }}
    >
      <div
        style={{
          fontSize: 16,
          fontWeight: 700,
          color: '#1f2937',
          marginBottom: 6,
        }}
      >
        {repo.name}
      </div>
      <div
        style={{
          fontSize: 13,
          color: '#6b7280',
          fontFamily: 'monospace',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}
      >
        {repo.path}
      </div>
    </div>
  );
};

export default RepoCard;
