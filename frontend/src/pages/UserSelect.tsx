import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import type { User } from '../types';
import { listUsers, createUser, selectUser } from '../api/client';

const styles: Record<string, React.CSSProperties> = {
  container: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: '100vh',
    padding: '2rem',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    backgroundColor: '#0f1117',
    color: '#e1e4e8',
  },
  title: {
    fontSize: '2rem',
    fontWeight: 700,
    marginBottom: '0.5rem',
  },
  subtitle: {
    fontSize: '1rem',
    color: '#8b949e',
    marginBottom: '2rem',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))',
    gap: '1rem',
    width: '100%',
    maxWidth: '720px',
    marginBottom: '2.5rem',
  },
  card: {
    backgroundColor: '#161b22',
    border: '1px solid #30363d',
    borderRadius: '8px',
    padding: '1.25rem',
    cursor: 'pointer',
    transition: 'border-color 0.2s, box-shadow 0.2s',
  },
  cardUsername: {
    fontSize: '1.1rem',
    fontWeight: 600,
    marginBottom: '0.5rem',
  },
  cardDate: {
    fontSize: '0.8rem',
    color: '#8b949e',
  },
  createSection: {
    width: '100%',
    maxWidth: '720px',
    backgroundColor: '#161b22',
    border: '1px solid #30363d',
    borderRadius: '8px',
    padding: '1.5rem',
  },
  createTitle: {
    fontSize: '1rem',
    fontWeight: 600,
    marginBottom: '1rem',
  },
  createForm: {
    display: 'flex',
    gap: '0.75rem',
  },
  input: {
    flex: 1,
    padding: '0.6rem 0.75rem',
    borderRadius: '6px',
    border: '1px solid #30363d',
    backgroundColor: '#0d1117',
    color: '#e1e4e8',
    fontSize: '0.9rem',
    outline: 'none',
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
  },
  buttonDisabled: {
    opacity: 0.6,
    cursor: 'not-allowed',
  },
  error: {
    color: '#f85149',
    fontSize: '0.85rem',
    marginTop: '0.75rem',
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

function UserSelect() {
  const navigate = useNavigate();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [newUsername, setNewUsername] = useState('');
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    fetchUsers();
  }, []);

  async function fetchUsers() {
    setLoading(true);
    setError(null);
    try {
      const data = await listUsers();
      setUsers(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users');
    } finally {
      setLoading(false);
    }
  }

  async function handleSelectUser(id: number) {
    try {
      await selectUser(id);
      navigate('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to select user');
    }
  }

  async function handleCreateUser(e: React.FormEvent) {
    e.preventDefault();
    const username = newUsername.trim();
    if (!username) return;

    setCreating(true);
    setError(null);
    try {
      const user = await createUser(username);
      await selectUser(user.id);
      navigate('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create user');
    } finally {
      setCreating(false);
    }
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  return (
    <div style={styles.container}>
      <h1 style={styles.title}>Claude Bot</h1>
      <p style={styles.subtitle}>Select a user to continue</p>

      {loading ? (
        <p style={styles.loading}>Loading users...</p>
      ) : (
        <>
          {users.length === 0 ? (
            <p style={styles.empty}>No users yet. Create one below to get started.</p>
          ) : (
            <div style={styles.grid}>
              {users.map((user) => (
                <div
                  key={user.id}
                  style={styles.card}
                  onClick={() => handleSelectUser(user.id)}
                  onMouseEnter={(e) => {
                    (e.currentTarget as HTMLDivElement).style.borderColor = '#58a6ff';
                    (e.currentTarget as HTMLDivElement).style.boxShadow = '0 0 0 1px #58a6ff';
                  }}
                  onMouseLeave={(e) => {
                    (e.currentTarget as HTMLDivElement).style.borderColor = '#30363d';
                    (e.currentTarget as HTMLDivElement).style.boxShadow = 'none';
                  }}
                >
                  <div style={styles.cardUsername}>{user.username}</div>
                  <div style={styles.cardDate}>Created {formatDate(user.created_at)}</div>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      <div style={styles.createSection}>
        <div style={styles.createTitle}>Create New User</div>
        <form style={styles.createForm} onSubmit={handleCreateUser}>
          <input
            style={styles.input}
            type="text"
            placeholder="Enter username"
            value={newUsername}
            onChange={(e) => setNewUsername(e.target.value)}
            disabled={creating}
          />
          <button
            style={{
              ...styles.button,
              ...(creating || !newUsername.trim() ? styles.buttonDisabled : {}),
            }}
            type="submit"
            disabled={creating || !newUsername.trim()}
          >
            {creating ? 'Creating...' : 'Create User'}
          </button>
        </form>
      </div>

      {error && <p style={styles.error}>{error}</p>}
    </div>
  );
}

export default UserSelect;
