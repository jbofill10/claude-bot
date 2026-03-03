export interface User {
  id: number;
  username: string;
  created_at: string;
}

export interface Repo {
  id: number;
  user_id: number;
  name: string;
  path: string;
  deploy_script: string;
  added_at: string;
}

export interface AvailableRepo {
  name: string;
  path: string;
}

export interface Task {
  id: number;
  user_id: number;
  repo_id: number;
  title: string;
  description: string;
  type: string;
  status: TaskStatus;
  branch_name: string;
  pr_number: number;
  plan_text: string;
  error_message: string;
  created_at: string;
  updated_at: string;
}

export type TaskStatus =
  | 'pending'
  | 'planning'
  | 'plan_review'
  | 'developing'
  | 'reviewing'
  | 'merging'
  | 'deploy_review'
  | 'deploying'
  | 'completed'
  | 'failed'
  | 'cancelled';

export interface TaskLog {
  id: number;
  task_id: number;
  stage: string;
  content: string;
  timestamp: string;
}

export interface Setting {
  key: string;
  value: string;
}

export interface WSMessage {
  type: 'output' | 'status';
  stage?: string;
  content?: string;
  status?: string;
  raw?: unknown;
}
