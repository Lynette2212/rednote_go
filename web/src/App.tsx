import { Route, Routes } from 'react-router-dom'
import AppHeader from './components/AppHeader'
import Home from './pages/Home'
import NoteDetail from './pages/NoteDetail'
import Publish from './pages/Publish'
import Login from './pages/Login'
import Register from './pages/Register'
import Profile from './pages/Profile'

export default function App() {
  return (
    <div className="app">
      <AppHeader />
      <main className="app-main">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/note/:id" element={<NoteDetail />} />
          <Route path="/publish" element={<Publish />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/user/:id" element={<Profile />} />
        </Routes>
      </main>
    </div>
  )
}
