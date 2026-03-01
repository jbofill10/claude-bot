import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Repo, AvailableRepo } from '../types';
import { listRepos, listAvailableRepos, createRepo } from '../api/client';

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
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '2rem',
  },
  title: {
    fontSize: '1.75rem',
    fontWeight: 700,
  },
  addButton: {
    padding: '0.6rem 1.25rem',
    borderRadius: '6px',
    border: 'none',
    backgroundColor: '#238636',
    color: '#ffffff',
    fontSize: '0.9rem',
    fontWeight: 600,
    cursor: 'pointer',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
    gap: '1rem',
  },
  card: {
    backgroundColor: '#161b22',
    border: '1px solid #30363d',
    borderRadius: '8px',
    padding: '1.25rem',
    cursor: 'pointer',
    transition: 'border-color 0.2s, box-shadow 0.2s',
  },
  cardName: {
    fontSize: '1.1rem',
    fontWeight: 600,
    marginBottom: '0.5rem',
  },
  cardPath: {
    fontSize: '0.8rem',
    color: '#8b949e',
    marginBottom: '0.75rem',
    wordBreak: 'break-all' as const,
  },
  cardTasks: {
    fontSize: '0.8rem',
    color: '#58a6ff',
  },
  modalOverlay: {
    position: 'fixed' as const,
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0, 0, 0, 0.6)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1000,
  },
  modal: {
    backgroundColor: '#161b22',
    border: '1px solid #30363d',
    borderRadius: '12px',
    padding: '1.5rem',
    width: '100%',
    maxWidth: '500px',
    maxHeight: '70vh',
    display: 'flex',
    flexDirection: 'column' as const,
  },
  modalTitle: {
    fontSize: '1.2rem',
    fontWeight: 600,
    marginBottom: '1rem',
  },
  modalList: {
    overflowY: 'auto' as const,
    flex: 1,
  },
  modalItem: {
    padding: '0.75rem',
    borderRadius: '6px',
    cursor: 'pointer',
    borderBottom: '1px solid #21262d',
    transition: 'background-color 0.15s',
  },
  modalItemName: {
    fontWeight: 600,
    fontSize: '0.95rem',
    marginBottom: '0.25rem',
  },
  modalItemPath: {
    fontSize: '0.8rem',
    color: '#8b949e',
    wordBreak: 'break-all' as const,
  },
  modalClose: {
    marginTop: '1rem',
    padding: '0.5rem 1rem',
    borderRadius: '6px',
    border: '1px solid #30363d',
    backgroundColor: 'transparent',
    color: '#e1e4e8',
    fontSize: '0.9rem',
    cursor: 'pointer',
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
    padding: '3rem 0',
  },
  modalEmpty: {
    color: '#8b949e',
    fontSize: '0.9rem',
    textAlign: 'center' as const,
    padding: '2rem 0',
  },
};

function Dashboard() {
  const navigate = useNavigate();
  const [repos, setRepos] = useState<Repo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [showModal, setShowModal] = useState(false);
  const [availableRepos, setAvailableRepos] = useState<AvailableRepo[]>([]);
  const [loadingAvailable, setLoadingAvailable] = useState(false);
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    fetchRepos();
  }, []);

  async function fetchRepos() {
    setLoading(true);
    setError(null);
    try {
      const data = await listRepos();
      setRepos(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load repositories');
    } finally {
      setLoading(false);
    }
  }

  async function handleOpenModal() {
    setShowModal(true);
    setLoadingAvailable(true);
    try {
      const data = await listAvailableRepos();
      setAvailableRepos(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load available repositories');
    } finally {
      setLoadingAvailable(false);
    }
  }

  async function handleAddRepo(path: string) {
    setAdding(true);
    setError(null);
    try {
      await createRepo(path);
      setShowModal(false);
      await fetchRepos();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add repository');
    } finally {
      setAdding(false);
    }
  }

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h1 style={styles.title}>Dashboard</h1>
        <button style={styles.addButton} onClick={handleOpenModal}>
          + Add Repository
        </button>
      </div>

      {error && <p style={styles.error}>{error}</p>}

      {loading ? (
        <p style={styles.loading}>Loading repositories...</p>
      ) : repos.length === 0 ? (
        <p style={styles.empty}>No repositories yet. Click "Add Repository" to get started.</p>
      ) : (
        <div style={styles.grid}>
          {repos.map((repo) => (
            <div
              key={repo.id}
              style={styles.card}
              onClick={() => navigate(`/repos/${repo.id}`)}
              onMouseEnter={(e) => {
                (e.currentTarget as HTMLDivElement).style.borderColor = '#58a6ff';
                (e.currentTarget as HTMLDivElement).style.boxShadow = '0 0 0 1px #58a6ff';
              }}
              onMouseLeave={(e) => {
                (e.currentTarget as HTMLDivElement).style.borderColor = '#30363d';
                (e.currentTarget as HTMLDivElement).style.boxShadow = 'none';
              }}
            >
              <div style={styles.cardName}>{repo.name}</div>
              <div style={styles.cardPath}>{repo.path}</div>
              <div style={styles.cardTasks}>-- tasks</div>
            </div>
          ))}
        </div>
      )}

      {showModal && (
        <div style={styles.modalOverlay} onClick={() => setShowModal(false)}>
          <div style={styles.modal} onClick={(e) => e.stopPropagation()}>
            <div style={styles.modalTitle}>Add Repository</div>
            <div style={styles.modalList}>
              {loadingAvailable ? (
                <p style={styles.loading}>Loading available repositories...</p>
              ) : availableRepos.length === 0 ? (
                <p style={styles.modalEmpty}>No available repositories found.</p>
              ) : (
                availableRepos.map((repo) => (
                  <div
                    key={repo.path}
                    style={{
                      ...styles.modalItem,
                      ...(adding ? { opacity: 0.6, pointerEvents: 'none' as const } : {}),
                    }}
                    onClick={() => handleAddRepo(repo.path)}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLDivElement).style.backgroundColor = '#21262d';
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLDivElement).style.backgroundColor = 'transparent';
                    }}
                  >
                    <div style={styles.modalItemName}>{repo.name}</div>
                    <div style={styles.modalItemPath}>{repo.path}</div>
                  </div>
                ))
              )}
            </div>
            <button style={styles.modalClose} onClick={() => setShowModal(false)}>
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default Dashboard;
