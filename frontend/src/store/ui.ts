import { create } from 'zustand'

interface UIState {
  activeWorkspaceId: string | null
  activeListId: string | null
  sidebarOpen: boolean
  setActiveWorkspace: (id: string) => void
  setActiveList: (id: string) => void
  toggleSidebar: () => void
}

export const useUIStore = create<UIState>((set) => ({
  activeWorkspaceId: null,
  activeListId: null,
  sidebarOpen: true,
  setActiveWorkspace: (id) => set({ activeWorkspaceId: id }),
  setActiveList: (id) => set({ activeListId: id }),
  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
}))
