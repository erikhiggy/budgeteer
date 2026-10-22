import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClientProvider } from '@tanstack/react-query';
import './styles/reset.css';
import './styles/tokens.css';
import './styles/global.css';
import { createQueryClient } from './api/hooks';
import { App } from './App';

const queryClient = createQueryClient();

createRoot(document.getElementById('root')!).render(
	<StrictMode>
		<QueryClientProvider client={queryClient}>
			<App />
		</QueryClientProvider>
	</StrictMode>,
);
