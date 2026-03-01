import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import type { Task, TaskStatus } from '../types';
import { getTask, retryTask, approveTask, rejectTask, approveDeployTask, skipDeployTask } from '../api/client';
import { useWebSocket } from '../hooks/useWebSocket';

// ---------- Stage Progress Component ----------

const STAGES: TaskStatus[] = [
  'pending',
  'planning',
  'plan_review',
  'developing',
  'reviewing',
  'merging',
  'deploy_review',
  'deploying',
  'completed',
];

const stageLabels: Record<string, string> = {
  pending: 'Pending',
  planning: 'Planning',
  plan_review: 'Review',
  developing: 'Developing',
  reviewing: 'Reviewing',
  merging: 'Merging',
  deploy_review: 'Deploy?',
  deploying: 'Deploying',
  completed: 'Completed',
  failed: 'Failed',
};

function StageProgress({ status }: { status: TaskStatus }) {
  const currentIndex = STAGES.indexOf(status);
  const isFailed = status === 'failed';

  return (
    <div style={{ marginBottom: '1.5rem' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
        {STAGES.map((stage, i) => {
          let bgColor = '#21262d';
          if (isFailed) {
            bgColor = i <= 0 ? '#f8514930' : '#21262d';
          } else if (i < currentIndex) {
            bgColor = '#238636';
          } else if (i === currentIndex) {
            bgColor = '#58a6ff';
          }

          return (
            <div key={stage} style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
              <div
                style={{
                  width: '100%',
                  height: '4px',
                  borderRadius: '2px',
                  backgroundColor: bgColor,
                  transition: 'background-color 0.3s',
                }}
              />
              <span
                style={{
                  fontSize: '0.65rem',
                  color: i === currentIndex ? '#58a6ff' : '#8b949e',
                  marginTop: '0.35rem',
                  fontWeight: i === currentIndex ? 600 : 400,
                }}
              >
                {stageLabels[stage]}
              </span>
            </div>
          );
        })}
      </div>
      {isFailed && (
        <div style={{ color: '#f85149', fontSize: '0.8rem', marginTop: '0.5rem', fontWeight: 600 }}>
          Task Failed
        </div>
      )}
    </div>
  );
}

// ---------- Plan View Component ----------

function PlanView({
  planText,
  onApprove,
  onReject,
  loading,
}: {
  planText: string;
  onApprove: () => void;
  onReject: (feedback: string) => void;
  loading: boolean;
}) {
  const [feedback, setFeedback] = useState('');
  const [showReject, setShowReject] = useState(false);

  return (
    <div
      style={{
        backgroundColor: '#161b22',
        border: '1px solid #30363d',
        borderRadius: '8px',
        padding: '1.25rem',
        marginBottom: '1.5rem',
      }}
    >
      <h3 style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '0.75rem' }}>
        Plan Review
      </h3>
      <pre
        style={{
          backgroundColor: '#0d1117',
          border: '1px solid #21262d',
          borderRadius: '6px',
          padding: '1rem',
          fontSize: '0.85rem',
          lineHeight: 1.5,
          overflowX: 'auto',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          color: '#e1e4e8',
          maxHeight: '400px',
          overflowY: 'auto',
          marginBottom: '1rem',
        }}
      >
        {planText}
      </pre>

      {!showReject ? (
        <div style={{ display: 'flex', gap: '0.75rem' }}>
          <button
            style={{
              padding: '0.6rem 1.5rem',
              borderRadius: '6px',
              border: 'none',
              backgroundColor: '#238636',
              color: '#ffffff',
              fontSize: '0.9rem',
              fontWeight: 600,
              cursor: loading ? 'not-allowed' : 'pointer',
              opacity: loading ? 0.6 : 1,
            }}
            onClick={onApprove}
            disabled={loading}
          >
            {loading ? 'Approving...' : 'Approve Plan'}
          </button>
          <button
            style={{
              padding: '0.6rem 1.5rem',
              borderRadius: '6px',
              border: '1px solid #f85149',
              backgroundColor: 'transparent',
              color: '#f85149',
              fontSize: '0.9rem',
              fontWeight: 600,
              cursor: 'pointer',
            }}
            onClick={() => setShowReject(true)}
            disabled={loading}
          >
            Request Changes
          </button>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
          <textarea
            style={{
              width: '100%',
              padding: '0.6rem 0.75rem',
              borderRadius: '6px',
              border: '1px solid #30363d',
              backgroundColor: '#0d1117',
              color: '#e1e4e8',
              fontSize: '0.9rem',
              outline: 'none',
              resize: 'vertical',
              minHeight: '80px',
              fontFamily: 'inherit',
              boxSizing: 'border-box',
            }}
            placeholder="Describe what changes you'd like..."
            value={feedback}
            onChange={(e) => setFeedback(e.target.value)}
          />
          <div style={{ display: 'flex', gap: '0.75rem' }}>
            <button
              style={{
                padding: '0.6rem 1.25rem',
                borderRadius: '6px',
                border: 'none',
                backgroundColor: '#f85149',
                color: '#ffffff',
                fontSize: '0.9rem',
                fontWeight: 600,
                cursor: loading || !feedback.trim() ? 'not-allowed' : 'pointer',
                opacity: loading || !feedback.trim() ? 0.6 : 1,
              }}
              onClick={() => onReject(feedback.trim())}
              disabled={loading || !feedback.trim()}
            >
              {loading ? 'Sending...' : 'Send Feedback'}
            </button>
            <button
              style={{
                padding: '0.6rem 1.25rem',
                borderRadius: '6px',
                border: '1px solid #30363d',
                backgroundColor: 'transparent',
                color: '#e1e4e8',
                fontSize: '0.9rem',
                cursor: 'pointer',
              }}
              onClick={() => {
                setShowReject(false);
                setFeedback('');
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// ---------- Deploy Review Component ----------

function DeployReview({
  onDeploy,
  onSkip,
  loading,
}: {
  onDeploy: () => void;
  onSkip: () => void;
  loading: boolean;
}) {
  return (
    <div
      style={{
        backgroundColor: '#161b22',
        border: '1px solid #30363d',
        borderRadius: '8px',
        padding: '1.25rem',
        marginBottom: '1.5rem',
      }}
    >
      <h3 style={{ fontSize: '1.1rem', fontWeight: 600, marginBottom: '0.75rem' }}>
        Deploy
      </h3>
      <p style={{ color: '#8b949e', fontSize: '0.9rem', marginBottom: '1rem', lineHeight: 1.5 }}>
        Task merged successfully. Would you like to deploy?
      </p>
      <div style={{ display: 'flex', gap: '0.75rem' }}>
        <button
          style={{
            padding: '0.6rem 1.5rem',
            borderRadius: '6px',
            border: 'none',
            backgroundColor: '#238636',
            color: '#ffffff',
            fontSize: '0.9rem',
            fontWeight: 600,
            cursor: loading ? 'not-allowed' : 'pointer',
            opacity: loading ? 0.6 : 1,
          }}
          onClick={onDeploy}
          disabled={loading}
        >
          {loading ? 'Starting...' : 'Deploy'}
        </button>
        <button
          style={{
            padding: '0.6rem 1.5rem',
            borderRadius: '6px',
            border: '1px solid #30363d',
            backgroundColor: 'transparent',
            color: '#e1e4e8',
            fontSize: '0.9rem',
            fontWeight: 600,
            cursor: loading ? 'not-allowed' : 'pointer',
            opacity: loading ? 0.6 : 1,
          }}
          onClick={onSkip}
          disabled={loading}
        >
          Skip
        </button>
      </div>
    </div>
  );
}

// ---------- Live Output Component ----------

function LiveOutput({ lines }: { lines: string[] }) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [lines]);

  if (lines.length === 0) return null;

  return (
    <div
      style={{
        backgroundColor: '#0d1117',
        border: '1px solid #21262d',
        borderRadius: '8px',
        padding: '1rem',
        maxHeight: '400px',
        overflowY: 'auto',
        marginBottom: '1.5rem',
        fontFamily: '"SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
        fontSize: '0.8rem',
        lineHeight: 1.6,
      }}
    >
      {lines.map((line, i) => (
        <div key={i} style={{ color: '#e1e4e8', whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
          {line}
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  );
}

// ---------- Main TaskDetail Component ----------

const statusColors: Record<string, string> = {
  pending: '#8b949e',
  planning: '#d29922',
  plan_review: '#58a6ff',
  developing: '#d29922',
  reviewing: '#bc8cff',
  merging: '#d29922',
  deploy_review: '#58a6ff',
  deploying: '#d29922',
  completed: '#3fb950',
  failed: '#f85149',
};

const styles: Record<string, React.CSSProperties> = {
  container: {
    minHeight: '100vh',
    padding: '2rem',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    backgroundColor: '#0f1117',
    color: '#e1e4e8',
    maxWidth: '960px',
    margin: '0 auto',
  },
  backLink: {
    color: '#58a6ff',
    fontSize: '0.85rem',
    cursor: 'pointer',
    marginBottom: '1rem',
    display: 'inline-block',
  },
  header: {
    marginBottom: '1.5rem',
  },
  titleRow: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.75rem',
    flexWrap: 'wrap' as const,
    marginBottom: '0.5rem',
  },
  taskTitle: {
    fontSize: '1.75rem',
    fontWeight: 700,
  },
  badge: {
    display: 'inline-block',
    padding: '0.2rem 0.6rem',
    borderRadius: '12px',
    fontSize: '0.75rem',
    fontWeight: 600,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.03em',
  },
  description: {
    fontSize: '0.95rem',
    color: '#8b949e',
    lineHeight: 1.5,
    marginBottom: '0.5rem',
  },
  meta: {
    fontSize: '0.8rem',
    color: '#6e7681',
  },
  section: {
    marginBottom: '1.5rem',
  },
  sectionTitle: {
    fontSize: '1.1rem',
    fontWeight: 600,
    marginBottom: '0.75rem',
  },
  errorBox: {
    backgroundColor: '#f8514915',
    border: '1px solid #f8514940',
    borderRadius: '8px',
    padding: '1.25rem',
    marginBottom: '1.5rem',
  },
  errorTitle: {
    color: '#f85149',
    fontWeight: 600,
    fontSize: '1rem',
    marginBottom: '0.5rem',
  },
  errorMessage: {
    color: '#f0883e',
    fontSize: '0.9rem',
    lineHeight: 1.5,
    whiteSpace: 'pre-wrap' as const,
    marginBottom: '1rem',
  },
  retryButton: {
    padding: '0.6rem 1.25rem',
    borderRadius: '6px',
    border: 'none',
    backgroundColor: '#f85149',
    color: '#ffffff',
    fontSize: '0.9rem',
    fontWeight: 600,
    cursor: 'pointer',
  },
  prLink: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '0.5rem',
    padding: '0.6rem 1rem',
    borderRadius: '6px',
    backgroundColor: '#238636',
    color: '#ffffff',
    fontSize: '0.9rem',
    fontWeight: 600,
    textDecoration: 'none',
    marginBottom: '1.5rem',
  },
  wsStatus: {
    fontSize: '0.75rem',
    display: 'inline-flex',
    alignItems: 'center',
    gap: '0.35rem',
    marginBottom: '1rem',
  },
  dot: {
    width: '6px',
    height: '6px',
    borderRadius: '50%',
    display: 'inline-block',
  },
  loading: {
    color: '#8b949e',
    fontSize: '1rem',
  },
  error: {
    color: '#f85149',
    fontSize: '0.85rem',
    marginBottom: '1rem',
  },
};

function TaskDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const taskId = Number(id);

  const [task, setTask] = useState<Task | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const { messages, connected } = useWebSocket(taskId);

  const fetchTask = useCallback(async () => {
    try {
      const data = await getTask(taskId);
      setTask(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load task');
    } finally {
      setLoading(false);
    }
  }, [taskId]);

  // Initial fetch
  useEffect(() => {
    fetchTask();
  }, [fetchTask]);

  // Poll every 5 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      fetchTask();
    }, 5000);
    return () => clearInterval(interval);
  }, [fetchTask]);

  // Collect output lines from WebSocket messages
  const outputLines: string[] = [];
  for (const msg of messages) {
    if (msg.type === 'output' && msg.content) {
      outputLines.push(msg.content);
    }
  }

  async function handleRetry() {
    setActionLoading(true);
    try {
      const updated = await retryTask(taskId);
      setTask(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to retry task');
    } finally {
      setActionLoading(false);
    }
  }

  async function handleApprove() {
    setActionLoading(true);
    try {
      const updated = await approveTask(taskId);
      setTask(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve task');
    } finally {
      setActionLoading(false);
    }
  }

  async function handleReject(feedback: string) {
    setActionLoading(true);
    try {
      const updated = await rejectTask(taskId, feedback);
      setTask(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reject task');
    } finally {
      setActionLoading(false);
    }
  }

  async function handleApproveDeploy() {
    setActionLoading(true);
    try {
      const updated = await approveDeployTask(taskId);
      setTask(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start deploy');
    } finally {
      setActionLoading(false);
    }
  }

  async function handleSkipDeploy() {
    setActionLoading(true);
    try {
      const updated = await skipDeployTask(taskId);
      setTask(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to skip deploy');
    } finally {
      setActionLoading(false);
    }
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  if (loading) {
    return (
      <div style={styles.container}>
        <p style={styles.loading}>Loading task...</p>
      </div>
    );
  }

  if (!task) {
    return (
      <div style={styles.container}>
        <span style={styles.backLink} onClick={() => navigate(-1)}>
          &larr; Back
        </span>
        <p style={styles.error}>Task not found.</p>
      </div>
    );
  }

  return (
    <div style={styles.container}>
      <span style={styles.backLink} onClick={() => navigate(`/repos/${task.repo_id}`)}>
        &larr; Back to Repository
      </span>

      <div style={styles.header}>
        <div style={styles.titleRow}>
          <h1 style={styles.taskTitle}>{task.title || task.description}</h1>
          <span
            style={{
              ...styles.badge,
              color: statusColors[task.status] || '#8b949e',
              backgroundColor: `${statusColors[task.status] || '#8b949e'}20`,
            }}
          >
            {task.status.replace('_', ' ')}
          </span>
        </div>
        {task.title && task.description && (
          <div style={styles.description}>{task.description}</div>
        )}
        <div style={styles.meta}>
          Created {formatDate(task.created_at)}
          {task.updated_at !== task.created_at && (
            <> &middot; Updated {formatDate(task.updated_at)}</>
          )}
          {task.branch_name && <> &middot; Branch: {task.branch_name}</>}
        </div>
      </div>

      {error && <p style={styles.error}>{error}</p>}

      {/* Stage Progress */}
      <StageProgress status={task.status} />

      {/* PR Link */}
      {task.pr_number > 0 && (
        <a
          href={`#pr-${task.pr_number}`}
          style={styles.prLink}
          title={`Pull Request #${task.pr_number}`}
        >
          Pull Request #{task.pr_number}
        </a>
      )}

      {/* Plan Review */}
      {task.status === 'plan_review' && task.plan_text && (
        <PlanView
          planText={task.plan_text}
          onApprove={handleApprove}
          onReject={handleReject}
          loading={actionLoading}
        />
      )}

      {/* Deploy Review */}
      {task.status === 'deploy_review' && (
        <DeployReview
          onDeploy={handleApproveDeploy}
          onSkip={handleSkipDeploy}
          loading={actionLoading}
        />
      )}

      {/* Error Section */}
      {task.status === 'failed' && (
        <div style={styles.errorBox}>
          <div style={styles.errorTitle}>Task Failed</div>
          {task.error_message && (
            <div style={styles.errorMessage}>{task.error_message}</div>
          )}
          <button
            style={{
              ...styles.retryButton,
              opacity: actionLoading ? 0.6 : 1,
              cursor: actionLoading ? 'not-allowed' : 'pointer',
            }}
            onClick={handleRetry}
            disabled={actionLoading}
          >
            {actionLoading ? 'Retrying...' : 'Retry Task'}
          </button>
        </div>
      )}

      {/* Live Output */}
      <div style={styles.section}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <h2 style={styles.sectionTitle}>Output</h2>
          <div style={styles.wsStatus}>
            <span
              style={{
                ...styles.dot,
                backgroundColor: connected ? '#3fb950' : '#8b949e',
              }}
            />
            <span style={{ color: connected ? '#3fb950' : '#8b949e' }}>
              {connected ? 'Connected' : 'Disconnected'}
            </span>
          </div>
        </div>
        <LiveOutput lines={outputLines} />
        {outputLines.length === 0 && (
          <p style={{ color: '#8b949e', fontSize: '0.85rem' }}>
            No output yet. Output will appear here in real time.
          </p>
        )}
      </div>
    </div>
  );
}

export default TaskDetail;
