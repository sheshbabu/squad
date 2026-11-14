import ApiClient from '../http/ApiClient.js';

const { createContext, useContext, useState, useEffect, useCallback } = React;

const AppContext = createContext(null);

export function AppProvider({ currentUser, children }) {
  const [bots, setBots] = useState([]);
  const [isBotsLoading, setIsBotsLoading] = useState(true);

  const refreshBots = useCallback(() => {
    return ApiClient.getBots()
      .then(setBots)
      .finally(() => setIsBotsLoading(false));
  }, []);

  useEffect(() => {
    refreshBots();
  }, [refreshBots]);

  const value = { currentUser, bots, isBotsLoading, refreshBots };

  return (
    <AppContext.Provider value={value}>
      {children}
    </AppContext.Provider>
  );
}

export function useAppContext() {
  return useContext(AppContext);
}
