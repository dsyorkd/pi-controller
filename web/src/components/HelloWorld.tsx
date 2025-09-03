import React from 'react';

interface HelloWorldProps {
  name?: string;
}

export const HelloWorld: React.FC<HelloWorldProps> = ({ name = 'Pi Controller' }) => {
  return (
    <div className="hello-world">
      <h1>Hello, {name}!</h1>
      <p>Welcome to the Pi Controller Management Dashboard</p>
      <div className="features">
        <h2>Features:</h2>
        <ul>
          <li>Manage Kubernetes clusters</li>
          <li>Monitor Pi nodes</li>
          <li>Control GPIO devices</li>
          <li>System monitoring</li>
        </ul>
      </div>
    </div>
  );
};

export default HelloWorld;
