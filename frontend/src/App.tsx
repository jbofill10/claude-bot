import { BrowserRouter, Routes, Route } from 'react-router-dom';
import UserSelect from './pages/UserSelect';
import Dashboard from './pages/Dashboard';
import RepoDetail from './pages/RepoDetail';
import TaskDetail from './pages/TaskDetail';
import './App.css';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<UserSelect />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/repos/:id" element={<RepoDetail />} />
        <Route path="/tasks/:id" element={<TaskDetail />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
