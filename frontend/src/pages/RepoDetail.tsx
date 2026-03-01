import { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import type { Repo, Task } from '../types';
import { listRepos, listTasks, createTask, updateRepoDeployScript } from '../api/client';

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

const typeColors: Record<string, string> = {
  feature: '#58a6ff',
  bugfix: '#f85149',
  refactor: '#bc8cff',
  default: '#8b949e',
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
  repoName: {
    fontSize: '1.75rem',
    fontWeight: 700,
    marginBottom: '0.25rem',
  },
  repoPath: {
    fontSize: '0.85rem',
    color: '#8b949e',
    wordBreak: 'break-all' as const,
  },
  newTaskSection: {
    backgroundColor: '#161b22',
    border: '1px solid #30363d',
    borderRadius: '8px',
    padding: '1.25rem',
    marginBottom: '2rem',
  },
  newTaskTitle: {
    fontSize: '1rem',
    fontWeight: 600,
    marginBottom: '0.75rem',
  },
  newTaskForm: {
    display: 'flex',
    gap: '0.75rem',
  },
  textarea: {
    flex: 1,
    padding: '0.6rem 0.75rem',
    borderRadius: '6px',
    border: '1px solid #30363d',
    backgroundColor: '#0d1117',
    color: '#e1e4e8',
    fontSize: '0.9rem',
    outline: 'none',
    resize: 'vertical' as const,
    minHeight: '40px',
    fontFamily: 'inherit',
  },
  button: {
    padding: '0.6rem 1.25rem',
    borderRadius: '6px',
    border: 'none',
    backgroundColor: '#238636',
    color: '#ffffff',
    fontSize: '0.9rem',
    fontWeight: 600,
    cursor: 'pointer',
    whiteSpace: 'nowrap' as const,
    alignSelf: 'flex-start',
  },
  buttonDisabled: {
    opacity: 0.6,
    cursor: 'not-allowed',
  },
  sectionTitle: {
    fontSize: '1.15rem',
    fontWeight: 600,
    marginBottom: '1rem',
  },
  taskList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '0.75rem',
  },
  taskCard: {
    backgroundColor: '#161b22',
    border: '1px solid #30363d',
    borderRadius: '8px',
    padding: '1rem 1.25rem',
    cursor: 'pointer',
    transition: 'border-color 0.2s, box-shadow 0.2s',
  },
  taskHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    marginBottom: '0.5rem',
    flexWrap: 'wrap' as const,
  },
  taskTitle: {
    fontSize: '1rem',
    fontWeight: 600,
    flex: 1,
    minWidth: 0,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap' as const,
  },
  badge: {
    display: 'inline-block',
    padding: '0.15rem 0.5rem',
    borderRadius: '12px',
    fontSize: '0.7rem',
    fontWeight: 600,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.03em',
  },
  taskTimestamps: {
    fontSize: '0.75rem',
    color: '#8b949e',
  },
  error: {
    color: '#f85149',
    fontSize: '0.85rem',
    marginBottom: '1rem',
  },
  loading: {
    color: '#8b949e',
    fontSize: '1rem',
  },
  empty: {
    color: '#8b949e',
    fontSize: '0.95rem',
    textAlign: 'center' as const,
    padding: '2rem 0',
  },
};

function RepoDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const repoId = Number(id);

  const [repo, setRepo] = useState<Repo | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [showNewTask, setShowNewTask] = useState(false);
  const [description, setDescription] = useState('');
  const [creating, setCreating] = useState(false);
  const [deployScript, setDeployScript] = useState('');
  const [savingDeploy, setSavingDeploy] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [repos, taskList] = await Promise.all([listRepos(), listTasks(repoId)]);
      const found = repos.find((r) => r.id === repoId);
      setRepo(found || null);
      if (found) setDeployScript(found.deploy_script || '');
      setTasks(taskList);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, [repoId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleCreateTask(e: React.FormEvent) {
    e.preventDefault();
    const desc = description.trim();
    if (!desc) return;

    setCreating(true);
    setError(null);
    try {
      await createTask(repoId, desc);
      setDescription('');
      setShowNewTask(false);
      await fetchData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create task');
    } finally {
      setCreating(false);
    }
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  async function handleSaveDeployScript() {
    if (!repo) return;
    setSavingDeploy(true);
    setError(null);
    try {
      const updated = await updateRepoDeployScript(repo.id, deployScript.trim());
      setRepo(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update deploy script');
    } finally {
      setSavingDeploy(false);
    }
  }

  function getTypeBadgeColor(type: string): string {
    return typeColors[type] || typeColors.default;
  }

  if (loading) {
    return (
      <div style={styles.container}>
        <p style={styles.loading}>Loading...</p>
      </div>
    );
  }

  if (!repo) {
    return (
      <div style={styles.container}>
        <span style={styles.backLink} onClick={() => navigate('/dashboard')}>
          &larr; Back to Dashboard
        </span>
        <p style={styles.error}>Repository not found.</p>
      </div>
    );
  }

  return (
    <div style={styles.container}>
      <span style={styles.backLink} onClick={() => navigate('/dashboard')}>
        &larr; Back to Dashboard
      </span>

      <div style={styles.header}>
        <h1 style={styles.repoName}>{repo.name}</h1>
        <div style={styles.repoPath}>{repo.path}</div>
      </div>

      {/* Deploy Script Configuration */}
      <div
        style={{
          backgroundColor: '#161b22',
          border: '1px solid #30363d',
          borderRadius: '8px',
          padding: '1.25rem',
          marginBottom: '2rem',
        }}
      >
        <div style={{ fontSize: '1rem', fontWeight: 600, marginBottom: '0.5rem' }}>
          Deploy Script
        </div>
        <p style={{ color: '#8b949e', fontSize: '0.8rem', marginBottom: '0.75rem' }}>
          Optional. If set, you will be prompted to deploy after each task is merged.
        </p>
        <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'center' }}>
          <input
            type="text"
            style={{
              flex: 1,
              padding: '0.5rem 0.75rem',
              borderRadius: '6px',
              border: '1px solid #30363d',
              backgroundColor: '#0d1117',
              color: '#e1e4e8',
              fontSize: '0.85rem',
              outline: 'none',
              fontFamily: '"SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace',
            }}
            placeholder="e.g. ./deploy.sh or make deploy"
            value={deployScript}
            onChange={(e) => setDeployScript(e.target.value)}
          />
          <button
            style={{
              ...styles.button,
              fontSize: '0.85rem',
              padding: '0.5rem 1rem',
              ...(savingDeploy || deployScript.trim() === (repo.deploy_script || '')
                ? styles.buttonDisabled
                : {}),
            }}
            onClick={handleSaveDeployScript}
            disabled={savingDeploy || deployScript.trim() === (repo.deploy_script || '')}
          >
            {savingDeploy ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>

      {error && <p style={styles.error}>{error}</p>}

      {!showNewTask ? (
        <button
          style={{ ...styles.button, marginBottom: '2rem' }}
          onClick={() => setShowNewTask(true)}
        >
          + New Task
        </button>
      ) : (
        <div style={styles.newTaskSection}>
          <div style={styles.newTaskTitle}>New Task</div>
          <form style={styles.newTaskForm} onSubmit={handleCreateTask}>
            <textarea
              style={styles.textarea}
              placeholder="Describe what you want to accomplish..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              disabled={creating}
            />
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              <button
                style={{
                  ...styles.button,
                  ...(creating || !description.trim() ? styles.buttonDisabled : {}),
                }}
                type="submit"
                disabled={creating || !description.trim()}
              >
                {creating ? 'Creating...' : 'Create'}
              </button>
              <button
                style={{
                  ...styles.button,
                  backgroundColor: 'transparent',
                  border: '1px solid #30363d',
                }}
                type="button"
                onClick={() => {
                  setShowNewTask(false);
                  setDescription('');
                }}
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      <h2 style={styles.sectionTitle}>Tasks</h2>

      {tasks.length === 0 ? (
        <p style={styles.empty}>No tasks yet. Create one to get started.</p>
      ) : (
        <div style={styles.taskList}>
          {tasks.map((task) => (
            <div
              key={task.id}
              style={styles.taskCard}
              onClick={() => navigate(`/tasks/${task.id}`)}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLDivElement).style.borderColor = '#58a6ff';
                (e.currentTarget as HTMLDivElement).style.boxShadow = '0 0 0 1px #58a6ff';
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLDivElement).style.borderColor = '#30363d';
                (e.currentTarget as HTMLDivElement).style.boxShadow = 'none';
              }}
            >
              <div style={styles.taskHeader}>
                <div style={styles.taskTitle}>
                  {task.title || task.description}
                </div>
                {task.type && (
                  <span
                    style={{
                      ...styles.badge,
                      color: getTypeBadgeColor(task.type),
                      border: `1px solid ${getTypeBadgeColor(task.type)}`,
                    }}
                  >
                    {task.type}
                  </span>
                )}
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
              <div style={styles.taskTimestamps}>
                Created {formatDate(task.created_at)}
                {task.updated_at !== task.created_at && (
                  <> &middot; Updated {formatDate(task.updated_at)}</>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default RepoDetail;
