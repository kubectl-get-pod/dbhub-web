import { create } from 'zustand'
import type { DatabaseUser, UserPrivilege } from '../types'

interface UserState {
  users: DatabaseUser[]
  privileges: Record<string, UserPrivilege[]>
  setUsers: (users: DatabaseUser[]) => void
  setPrivileges: (user: string, privs: UserPrivilege[]) => void
}

export const useUserStore = create<UserState>((set) => ({
  users: [],
  privileges: {},
  setUsers: (users) => set({ users }),
  setPrivileges: (user, privs) => set((s) => ({ privileges: { ...s.privileges, [user]: privs } })),
}))
