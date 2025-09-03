import './App.css';
import HelloWorld from './components/HelloWorld';
import { useAppStore } from './store/useAppStore';

function App() {
  const { isLoading, error } = useAppStore((state) => ({
    isLoading: state.isLoading,
    error: state.error,
  }));

  if (error) {
    return (
      <div className="app-error">
        <h1>Error</h1>
        <p>{error}</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="app-loading">
        <p>Loading...</p>
      </div>
    );
  }

  return (
    <div className="app">
      <HelloWorld />
    </div>
  );
}

export default App;
