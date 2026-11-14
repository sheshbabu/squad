import { ArrowDownIcon, PencilIcon, InfoIcon, TrashIcon } from '../../commons/components/Icon.jsx';
import './ThreadMenu.css';

const { useState, useRef, useEffect } = React;

export default function ThreadMenu({ title, onRenameClick, onDetailsClick, onDeleteClick }) {
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef(null);

  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsDropdownOpen(false);
      }
    }

    document.addEventListener("click", handleClickOutside);
    return () => document.removeEventListener("click", handleClickOutside);
  }, []);

  function handleTriggerClick() {
    setIsDropdownOpen(previous => !previous);
  }

  function handleItemClick(action) {
    setIsDropdownOpen(false);
    action();
  }

  return (
    <div ref={dropdownRef} className={`thread-menu ${isDropdownOpen ? "is-open" : ""}`}>
      <div className="thread-menu-trigger" onClick={handleTriggerClick}>
        <span className="thread-menu-title">{title}</span>
        <ArrowDownIcon />
      </div>
      <div className="thread-menu-dropdown">
        <ul className="thread-menu-options">
          <li className="thread-menu-option" onClick={() => handleItemClick(onRenameClick)}>
            <PencilIcon />
            <span>Rename</span>
          </li>
          <li className="thread-menu-option" onClick={() => handleItemClick(onDetailsClick)}>
            <InfoIcon />
            <span>Details</span>
          </li>
          <li className="thread-menu-option is-destructive" onClick={() => handleItemClick(onDeleteClick)}>
            <TrashIcon />
            <span>Delete</span>
          </li>
        </ul>
      </div>
    </div>
  );
}
