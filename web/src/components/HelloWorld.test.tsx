import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import HelloWorld from './HelloWorld';

describe('HelloWorld Component', () => {
  it('renders with default name', () => {
    render(<HelloWorld />);

    expect(screen.getByText('Hello, Pi Controller!')).toBeInTheDocument();
    expect(
      screen.getByText('Welcome to the Pi Controller Management Dashboard')
    ).toBeInTheDocument();
  });

  it('renders with custom name', () => {
    const customName = 'Custom Controller';
    render(<HelloWorld name={customName} />);

    expect(screen.getByText(`Hello, ${customName}!`)).toBeInTheDocument();
  });

  it('renders all feature list items', () => {
    render(<HelloWorld />);

    const features = [
      'Manage Kubernetes clusters',
      'Monitor Pi nodes',
      'Control GPIO devices',
      'System monitoring',
    ];

    features.forEach((feature) => {
      expect(screen.getByText(feature)).toBeInTheDocument();
    });
  });

  it('renders features heading', () => {
    render(<HelloWorld />);

    expect(screen.getByText('Features:')).toBeInTheDocument();
  });

  it('has correct structure', () => {
    render(<HelloWorld />);

    const container = screen.getByText('Hello, Pi Controller!').closest('.hello-world');
    expect(container).toBeInTheDocument();

    const featuresList = screen.getByRole('list');
    expect(featuresList).toBeInTheDocument();

    const listItems = screen.getAllByRole('listitem');
    expect(listItems).toHaveLength(4);
  });
});
