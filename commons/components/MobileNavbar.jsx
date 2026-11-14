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

  return (
    <div className="mobile-navbar-container">
      <div className="mobile-navbar">
        <div className="mobile-navbar-button" onClick={toggleSidebar}>
          <BotIcon />
          Bots
        </div>
        <div className="mobile-navbar-button" onClick={handleSearchClick}>
          <SearchIcon />
          Search
        </div>
        <Link className="mobile-navbar-button" to={`/bots/${currentBotId}/new`}>
          <NewIcon />
          New
        </Link>
      </div>
    </div>
  );
}
