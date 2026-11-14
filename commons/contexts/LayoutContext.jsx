import isMobile from '../utils/isMobile.js';

const { createContext, useContext, useState, useCallback } = React;

const LayoutContext = createContext(null);

export function LayoutProvider({ children }) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(!isMobile());
  const [isChatExpanded, setIsChatExpanded] = useState(false);

  const toggleSidebar = useCallback(() => {
    setIsSidebarOpen(previous => !previous);
  }, []);

  const closeSidebar = useCallback(() => {
    setIsSidebarOpen(false);
  }, []);

  const toggleChatExpanded = useCallback(() => {
    setIsChatExpanded(previous => !previous);
  }, []);

  const value = { isSidebarOpen, isChatExpanded, toggleSidebar, closeSidebar, toggleChatExpanded };

  return (
    <LayoutContext.Provider value={value}>
      {children}
    </LayoutContext.Provider>
  );
}

export function useLayout() {
  return useContext(LayoutContext);
}
