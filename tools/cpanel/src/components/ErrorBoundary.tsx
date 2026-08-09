import { Component, type ReactNode } from 'react';

interface Props { children: ReactNode; }
interface State { error: Error | null; }

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error) { return { error }; }

  render() {
    if (this.state.error) {
      return (
        <div style={{ padding: 40, color: '#f85149', fontFamily: 'monospace', background: '#0d1117', minHeight: '100vh' }}>
          <h2 style={{ fontSize: 20, marginBottom: 16 }}>Application Error</h2>
          <pre style={{ fontSize: 13, whiteSpace: 'pre-wrap', background: '#161b22', padding: 16, borderRadius: 8, border: '1px solid #30363d' }}>
            {this.state.error.message}
            {'\n\n'}
            {this.state.error.stack}
          </pre>
        </div>
      );
    }
    return this.props.children;
  }
}
