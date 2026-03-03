import type { User, Repo, AvailableRepo, Task, TaskLog, Setting } from '../types';

const BASE = '/api';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || res.statusText);
  }
  return res.json();
}

// Users
export const listUsers = () => request<User[]>('/users');
export const createUser = (username: string) =>
  request<User>('/users', { method: 'POST', body: JSON.stringify({ username }) });
export const selectUser = (id: number) =>
  request<User>(`/users/${id}/select`, { method: 'POST' });

// Repos
export const listRepos = () => request<Repo[]>('/repos');
export const createRepo = (path: string) =>
  request<Repo>('/repos', { method: 'POST', body: JSON.stringify({ path }) });
export const deleteRepo = (id: number) =>
  request<{ status: string }>(`/repos/${id}`, { method: 'DELETE' });
export const listAvailableRepos = () => request<AvailableRepo[]>('/repos/available');

// Tasks
export const listTasks = (repoId: number) => request<Task[]>(`/repos/${repoId}/tasks`);
export const createTask = (repoId: number, description: string) =>
  request<Task>(`/repos/${repoId}/tasks`, {
    method: 'POST',
    body: JSON.stringify({ description }),
  });
export const getTask = (id: number) => request<Task>(`/tasks/${id}`);
export const approveTask = (id: number) =>
  request<Task>(`/tasks/${id}/approve`, { method: 'POST' });
export const rejectTask = (id: number, feedback: string) =>
  request<Task>(`/tasks/${id}/reject`, {
    method: 'POST',
    body: JSON.stringify({ feedback }),
  });
export const retryTask = (id: number) =>
  request<Task>(`/tasks/${id}/retry`, { method: 'POST' });
export const approveDeployTask = (id: number) =>
  request<Task>(`/tasks/${id}/approve-deploy`, { method: 'POST' });
export const skipDeployTask = (id: number) =>
  request<Task>(`/tasks/${id}/skip-deploy`, { method: 'POST' });
export const cancelTask = (id: number) =>
  request<Task>(`/tasks/${id}/cancel`, { method: 'POST' });
export const getTaskLogs = (id: number) => request<TaskLog[]>(`/tasks/${id}/logs`);

// Repos - deploy script
export const updateRepoDeployScript = (id: number, deployScript: string) =>
  request<Repo>(`/repos/${id}/deploy-script`, {
    method: 'PUT',
    body: JSON.stringify({ deploy_script: deployScript }),
  });

// Settings
export const listSettings = () => request<Setting[]>('/settings');
export const updateSetting = (key: string, value: string) =>
  request<Setting>(`/settings/${key}`, {
    method: 'PUT',
    body: JSON.stringify({ value }),
  });
