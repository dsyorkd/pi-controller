# Pi Controller Web Dashboard

This is the React-based web dashboard for the Pi Controller project, providing a user interface for cluster and node management.

## Features Implemented

### Task 31 - Cluster Overview and Node Management UI

✅ **Completed Components:**

1. **React Router Navigation**
   - Added `react-router-dom` for client-side routing
   - Implemented nested route structure with layout component

2. **Dashboard Components**
   - `Dashboard` - Main dashboard page combining cluster overview and node management
   - `ClusterOverview` - Grid view of cluster status cards with metrics
   - `NodeManagement` - Comprehensive node list with filtering and management controls

3. **Detail Pages**
   - `ClusterDetail` - Detailed cluster view with node list and management
   - `NodeDetail` - Individual node details with GPIO devices and cluster assignment
   - `NodesPage` - Dedicated node management page

4. **Layout & Navigation**
   - `Navigation` - Top navigation bar with active state indication
   - `Layout` - Main layout wrapper with navigation and content area
   - Responsive design with mobile menu support

5. **Custom Hooks**
   - `useClusters` - Cluster data fetching and management
   - `useNodes` - Node data fetching and management with filtering helpers

6. **Common Components**
   - `LoadingSpinner` - Reusable loading indicator
   - `StatusBadge` - Status visualization for clusters and nodes

7. **Integration**
   - Full integration with existing Zustand store (`useAppStore`)
   - Uses existing API service layer (`apiService`)
   - Maintains existing TypeScript types

## File Structure

```
src/
├── components/
│   ├── common/
│   │   ├── LoadingSpinner.tsx
│   │   └── StatusBadge.tsx
│   ├── dashboard/
│   │   ├── ClusterOverview.tsx
│   │   └── NodeManagement.tsx
│   └── layout/
│       ├── Layout.tsx
│       └── Navigation.tsx
├── hooks/
│   ├── useClusters.ts
│   └── useNodes.ts
├── pages/
│   ├── Dashboard.tsx
│   ├── ClusterDetail.tsx
│   ├── NodeDetail.tsx
│   └── NodesPage.tsx
├── services/api.ts (existing)
├── store/useAppStore.ts (existing)
├── types/index.ts (existing)
├── App.tsx (updated with routing)
└── index.css (updated with Tailwind-like utilities)
```

## Routes

- `/` - Main dashboard with cluster overview and node management
- `/clusters/:id` - Individual cluster detail page
- `/nodes` - Dedicated node management page
- `/nodes/:id` - Individual node detail page

## Key Features

### Cluster Management
- Visual cluster status cards with metrics
- Cluster health indicators (active/inactive/error)
- Node count and online status per cluster
- Direct navigation to cluster details

### Node Management
- Comprehensive node table with filtering
- Node provisioning/deprovisioning controls
- Status tracking (online/offline/provisioning/error)
- Hardware information display
- GPIO device visualization
- Cluster assignment management

### Interactive Features
- Real-time data fetching with loading states
- Error handling with retry capabilities
- Responsive design for mobile and desktop
- Intuitive navigation with breadcrumbs
- Status-based filtering and searching

## Styling

The interface uses a custom CSS utility system inspired by Tailwind CSS, providing:
- Consistent spacing and typography
- Professional color scheme
- Responsive grid layouts
- Interactive hover states
- Smooth transitions

## API Integration

The dashboard integrates seamlessly with the existing Pi Controller API:
- Cluster CRUD operations
- Node discovery and management
- Provisioning workflows
- GPIO device monitoring
- System health checks

## Development

To run the dashboard:

```bash
# Install dependencies (including new react-router-dom)
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

## Next Steps

The dashboard provides a solid foundation for cluster and node management. Future enhancements could include:

1. Real-time WebSocket updates for live status monitoring
2. Cluster creation forms and wizards
3. Advanced node filtering and search
4. GPIO device control interface
5. System metrics and monitoring charts
6. User authentication and role management
7. Bulk operations for multiple nodes/clusters