import Sidebar from '../../commons/components/Sidebar.jsx';
import EmptyState from '../../commons/components/EmptyState.jsx';
import Spinner from '../../commons/components/Spinner.jsx';
import Button from '../../commons/components/Button.jsx';
import { openModal } from '../../commons/components/Modal.jsx';
import { BotIcon } from '../../commons/components/Icon.jsx';
import navigateTo from '../../commons/utils/navigateTo.js';
import { useAppContext } from '../../commons/contexts/AppContext.jsx';
import { LayoutProvider } from '../../commons/contexts/LayoutContext.jsx';
import BotSettingsModal from '../bots/BotSettingsModal.jsx';
import './HomePage.css';

const { useEffect } = React;

export default function HomePage() {
  const { bots, isBotsLoading, refreshBots } = useAppContext();

  useEffect(() => {
    if (bots.length > 0) {
      navigateTo(`/bots/${bots[0].botId}`, false, true);
    }
  }, [bots]);

  function handleCreateClick() {
    openModal(<BotSettingsModal onSaved={handleBotCreated} />);
  }

  function handleBotCreated(bot) {
    refreshBots();
    navigateTo(`/bots/${bot.botId}`);
  }

  let content = <Spinner />;
  if (!isBotsLoading && bots.length === 0) {
    const actionButton = <Button variant="primary" onClick={handleCreateClick}>Create a bot</Button>;
    content = <EmptyState icon={<BotIcon />} title="No bots yet" description="A bot is an assistant with its own model, memories, files and schedules." actionButton={actionButton} />;
  }

  return (
    <LayoutProvider>
      <div className="page-container">
        <Sidebar />
        <div className="home-page">{content}</div>
      </div>
    </LayoutProvider>
  );
}
