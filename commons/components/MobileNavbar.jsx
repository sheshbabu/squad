import Link from './Link.jsx';
import SearchMenu from "../../features/search/SearchMenu.jsx";
import { openModal } from "./Modal.jsx";
import { BotIcon, SearchIcon, NewIcon } from "./Icon.jsx";
import { useLayout } from "../contexts/LayoutContext.jsx";
import "./MobileNavbar.css";

export default function MobileNavbar({ currentBotId }) {
  const { toggleSidebar } = useLayout();

  function handleSearchClick() {
    openModal(<SearchMenu />);
  }

  let newChatLink = null;
  if (currentBotId !== undefined) {
    newChatLink = (
      <Link className="mobile-navbar-button" to={`/bots/${currentBotId}/new`}>
        <NewIcon />
        New
      </Link>
    );
  }

  return (
    <div className="mobile-navbar">
      <div className="mobile-navbar-bar">
        <div className="mobile-navbar-button" onClick={toggleSidebar}>
          <BotIcon />
          Bots
        </div>
        <div className="mobile-navbar-button" onClick={handleSearchClick}>
          <SearchIcon />
          Search
        </div>
        {newChatLink}
      </div>
    </div>
  );
}
