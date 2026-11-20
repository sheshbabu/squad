import Router from './commons/components/Router.jsx';
import Route from './commons/components/Route.jsx';
import Spinner from './commons/components/Spinner.jsx';
import OfflineIndicator from './commons/components/OfflineIndicator.jsx';
import { openModal, closeModal, isModalOpen } from './commons/components/Modal.jsx';
import useAuth from './commons/auth/useAuth.jsx';
import { AppProvider } from './commons/contexts/AppContext.jsx';
import LoginPage from './features/users/LoginPage.jsx';
import HomePage from './features/home/HomePage.jsx';
import BotPage from './features/bots/BotPage.jsx';
import InboxPage from './features/inbox/InboxPage.jsx';
import SearchMenu from './features/search/SearchMenu.jsx';
import ThemePreferences from './commons/preferences/ThemePreferences.js';

ThemePreferences.applyTheme();

document.addEventListener("keydown", event => {
  if ((event.metaKey || event.ctrlKey) && event.key === "k") {
    event.preventDefault();
    openModal(<SearchMenu />);
    return;
  }

  if (event.key === "Escape" && isModalOpen()) {
    event.preventDefault();
    closeModal();
  }
});

function App() {
  const { isLoading, currentUser, shouldShowLogin, shouldShowOnboarding } = useAuth();

  if (isLoading) {
    return <div className="spinner-container"><Spinner /></div>;
  }

  if (shouldShowOnboarding) {
    return <LoginPage isOnboarding />;
  }

  if (shouldShowLogin || currentUser === null) {
    return <LoginPage />;
  }

  return (
    <AppProvider currentUser={currentUser}>
      <OfflineIndicator />
      <Router>
        <Route path="/" component={HomePage} />
        <Route path="/inbox" component={InboxPage} />
        <Route path="/inbox/threads/:threadId" component={InboxPage} />
        <Route path="/bots/:botId" component={BotPage} />
        <Route path="/bots/:botId/new" component={BotPage} />
        <Route path="/bots/:botId/threads/:threadId" component={BotPage} />
        <Route path="/bots/:botId/files" component={BotPage} />
        <Route path="/bots/:botId/files/:fileName" component={BotPage} />
      </Router>
    </AppProvider>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App />);
