import { Routes, Route, NavLink } from 'react-router-dom'
import InventoryList from './pages/InventoryList'
import InventoryDetail from './pages/InventoryDetail'
import EvaluationDetail from './pages/EvaluationDetail'
import Sources from './pages/Sources'

function Nav() {
  const link = ({ isActive }: { isActive: boolean }) =>
    `px-3 py-2 rounded text-sm font-medium transition-colors ${
      isActive
        ? 'bg-slate-700 text-white'
        : 'text-slate-300 hover:bg-slate-700 hover:text-white'
    }`
  return (
    <nav className="bg-slate-900 text-white">
      <div className="max-w-7xl mx-auto px-4 flex items-center gap-1 h-14">
        <span className="text-white font-bold text-lg mr-6">TrueState</span>
        <NavLink to="/" end className={link}>Inventories</NavLink>
        <NavLink to="/sources" className={link}>Sources</NavLink>
      </div>
    </nav>
  )
}

export default function App() {
  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      <Nav />
      <main className="max-w-7xl mx-auto px-4 py-8">
        <Routes>
          <Route path="/" element={<InventoryList />} />
          <Route path="/inventories/:id" element={<InventoryDetail />} />
          <Route path="/evaluations/:id" element={<EvaluationDetail />} />
          <Route path="/sources" element={<Sources />} />
        </Routes>
      </main>
    </div>
  )
}
