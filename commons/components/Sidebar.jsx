import Link from './Link.jsx';
import BotAvatar from './BotAvatar.jsx';
import SearchMenu from "../../features/search/SearchMenu.jsx";
import SettingsModal from "../../features/users/SettingsModal.jsx";
import BotSettingsModal from "../../features/bots/BotSettingsModal.jsx";
import { openModal } from "./Modal.jsx";
import { SearchIcon, SettingsIcon, PencilIcon } from "./Icon.jsx";
import navigateTo from "../utils/navigateTo.js";
import isMobile from "../utils/isMobile.js";
import { useAppContext } from "../contexts/AppContext.jsx";
import { useLayout } from "../contexts/LayoutContext.jsx";
import "./Sidebar.css";

export default function Sidebar({ currentBotId }) {
  const { bots, refreshBots } = useAppContext();
  const { isSidebarOpen, closeSidebar } = useLayout();

  function handleBotCreated(bot) {
    refreshBots();
    navigateTo(`/bots/${bot.botId}`);
  }

  function handleBotDeleted() {
    refreshBots();
    navigateTo("/");
  }

  function handleSearchClick() {
    openModal(<SearchMenu />);
  }

  function handleSettingsClick() {
    openModal(<SettingsModal bots={bots} onBotCreated={handleBotCreated} onBotSaved={refreshBots} onBotDeleted={handleBotDeleted} />);
  }

  function handleBotEditClick(bot) {
    openModal(<BotSettingsModal bot={bot} onSaved={refreshBots} onDeleted={handleBotDeleted} />);
  }

  function handleBotClick() {
    if (isMobile()) {
      closeSidebar();
    }
  }

  function handleBackdropClick() {
    closeSidebar();
  }

  const botLinks = bots.map(bot => (
    <div key={bot.botId} className="sidebar-bot-row">
      <Link
        className={`sidebar-button sidebar-bot-link color-${bot.color}`}
        activeClassName="is-active"
        isActive={bot.botId === currentBotId}
        to={`/bots/${bot.botId}`}
        onClick={handleBotClick}>
        <BotAvatar name={bot.icon} />
        <span className="sidebar-bot-name">{bot.name}</span>
      </Link>
      <button type="button" className="sidebar-bot-edit" title="Bot settings" onClick={() => handleBotEditClick(bot)}><PencilIcon /></button>
    </div>
  ));

  return (
    <>
      <div className={`sidebar-backdrop-container ${isSidebarOpen ? 'is-open' : ''}`} onClick={handleBackdropClick}>&nbsp;</div>
      <div className={`sidebar-container ${isSidebarOpen ? 'is-open' : ''}`}>
        <div className="sidebar-fixed">
          <div className="sidebar-button search" onClick={handleSearchClick}>
            <SearchIcon />
            Search
          </div>
          <div className="sidebar-button settings" onClick={handleSettingsClick}>
            <SettingsIcon />
            Settings
          </div>
        </div>

        <div className="sidebar-scrollable">
          <div className="sidebar-section">
            <div className="sidebar-section-title">Bots</div>
            {botLinks}
          </div>
        </div>
      </div>
    </>
  );
}
