package manager

type opt func(m *Manager)

func WithoutGracefulShutdown() opt {
	return func(m *Manager) {
		m.useGracefulShutdown = false
	}
}
