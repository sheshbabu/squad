import { ModalBackdrop, ModalContainer, closeModal } from '../../commons/components/Modal.jsx';
import { SearchIcon } from '../../commons/components/Icon.jsx';
import BotAvatar from '../../commons/components/BotAvatar.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import navigateTo from '../../commons/utils/navigateTo.js';
import SearchHistory from '../../commons/preferences/SearchHistory.js';
import RecentSearchQuery from './RecentSearchQuery.js';
import splitHighlights from './highlights.js';
import './SearchMenu.css';

const { useState, useEffect, useRef } = React;

export default function SearchMenu() {
  const [query, setQuery] = useState(RecentSearchQuery.get());
  const [results, setResults] = useState([]);
  const [hasSearched, setHasSearched] = useState(false);
  const [selectedItem, setSelectedItem] = useState(null);
  const [searchHistory, setSearchHistory] = useState([]);
  const [bots, setBots] = useState([]);

  const inputRef = useRef(null);
  const debounceTimerRef = useRef(null);
  const isPointerActiveRef = useRef(false);

  useEffect(() => {
    if (inputRef.current !== null) {
      inputRef.current.focus();
      // The cursor goes to the end, not the start, when the modal reopens with a recalled query.
      const length = inputRef.current.value.length;
      inputRef.current.setSelectionRange(length, length);
    }

    const history = SearchHistory.getItems();
    setSearchHistory(history);
    if (history.length > 0) {
      setSelectedItem(history[0]);
    }

    // The modal renders in its own root outside the app's context, so it loads bots itself for their icons and colors.
    ApiClient.getBots().then(setBots);

    if (query.trim() !== "") {
      runSearch(query);
    }
  }, []);

  useEffect(() => {
    RecentSearchQuery.set(query);
  }, [query]);

  function handleChange(event) {
    const value = event.target.value;
    setQuery(value);

    if (debounceTimerRef.current !== null) {
      clearTimeout(debounceTimerRef.current);
    }

    if (value.trim() === "") {
      setResults([]);
      setSelectedItem(searchHistory.length > 0 ? searchHistory[0] : null);
      setHasSearched(false);
      return;
    }

    setHasSearched(false);
    debounceTimerRef.current = setTimeout(() => runSearch(value), 200);
  }

  function runSearch(value) {
    ApiClient.search(value)
      .then(found => {
        setResults(found);
        setHasSearched(true);
        if (found.length > 0) {
          setSelectedItem(found[0]);
        }
      })
      .catch(() => { });
  }

  function handleKeyDown(event) {
    const items = query.trim() === "" ? searchHistory : results;
    const index = items.findIndex(item => item.threadId === selectedItem?.threadId);

    if (event.key === "ArrowDown") {
      event.preventDefault();
      isPointerActiveRef.current = false;
      setSelectedItem(items[index + 1 < items.length ? index + 1 : 0] ?? null);
      return;
    }

    if (event.key === "ArrowUp") {
      event.preventDefault();
      isPointerActiveRef.current = false;
      setSelectedItem(items[index - 1 >= 0 ? index - 1 : items.length - 1] ?? null);
      return;
    }

    if (event.key === "Enter" && selectedItem !== null) {
      event.preventDefault();
      handleResultClick(selectedItem);
    }
  }

  // Hover only moves the selection after the mouse actually moves, so keyboard scrolling doesn't fight a resting cursor.
  function handleResultsMouseMove() {
    isPointerActiveRef.current = true;
  }

  function handleResultMouseEnter(item) {
    if (!isPointerActiveRef.current) {
      return;
    }
    setSelectedItem(item);
  }

  function handleResultClick(item) {
    SearchHistory.saveItem(item);
    closeModal();
    navigateTo(`/bots/${item.botId}/threads/${item.threadId}`);
  }

  function renderItems(items, keyPrefix) {
    return items.map(item => (
      <SearchResultItem
        key={`${keyPrefix}-${item.threadId}`}
        item={item}
        bot={bots.find(bot => bot.botId === item.botId)}
        isSelected={selectedItem !== null && item.threadId === selectedItem.threadId}
        onClick={() => handleResultClick(item)}
        onMouseEnter={() => handleResultMouseEnter(item)} />
    ));
  }

  let historySection = null;
  let chatsSection = null;
  let emptySection = null;

  if (query.trim() === "") {
    if (searchHistory.length > 0) {
      historySection = (
        <div className="search-menu-section">
          <h4 className="search-menu-section-title">Recent</h4>
          {renderItems(searchHistory, "history")}
        </div>
      );
    }
  } else if (results.length > 0) {
    chatsSection = (
      <div className="search-menu-section">
        <h4 className="search-menu-section-title">Chats</h4>
        {renderItems(results, "chat")}
      </div>
    );
  } else if (hasSearched) {
    emptySection = <div className="search-menu-empty">No results for "{query}"</div>;
  }

  return (
    <ModalBackdrop onClose={closeModal} isCentered={false}>
      <ModalContainer className="search-menu">
        <div className="search-menu-input-container">
          <SearchIcon />
          <input type="text" placeholder="Search..." spellCheck="false" ref={inputRef} value={query} onChange={handleChange} onKeyDown={handleKeyDown} />
        </div>
        <div className="search-menu-results-container" onMouseMove={handleResultsMouseMove}>
          {historySection}
          {chatsSection}
          {emptySection}
        </div>
      </ModalContainer>
    </ModalBackdrop>
  );
}

function SearchResultItem({ item, bot, isSelected, onClick, onMouseEnter }) {
  const subtitleSegments = splitHighlights(item.snippet);

  return (
    <div className={`search-menu-result-item ${isSelected ? "is-selected" : ""}`} onClick={onClick} onMouseEnter={onMouseEnter}>
      <span className={`search-menu-result-item-icon color-${bot?.color ?? "gray"}`}><BotAvatar name={bot?.icon} /></span>
      <div className="search-menu-result-item-content">
        <p className="search-menu-result-item-title">{item.title}</p>
        <p className="search-menu-result-item-subtitle"><HighlightedText segments={subtitleSegments} /></p>
      </div>
    </div>
  );
}

function HighlightedText({ segments }) {
  return segments.map((segment, index) => {
    if (segment.isMatch) {
      return <mark key={index}>{segment.text}</mark>;
    }
    return segment.text;
  });
}
