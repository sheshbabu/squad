import { CheckIcon, ChevronDownIcon } from './Icon.jsx';
import './Dropdown.css';

const { useState, useEffect, useRef } = React;

// Options are { value, label, description }, where description is an optional second line such as an id.
export default function Dropdown({ id, label, hint = "", options, value, placeholder = "Select", isSearchable = false, onChange }) {
  const [isOpen, setIsOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [highlightedIndex, setHighlightedIndex] = useState(0);
  const containerRef = useRef(null);
  const menuRef = useRef(null);

  const normalizedQuery = query.trim().toLowerCase();
  const filteredOptions = options.filter(option => {
    if (normalizedQuery === "") {
      return true;
    }
    const text = `${option.label} ${option.description ?? ""}`.toLowerCase();
    return text.includes(normalizedQuery);
  });

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    function handleDocumentMouseDown(event) {
      if (!containerRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    }

    document.addEventListener("mousedown", handleDocumentMouseDown);
    return () => document.removeEventListener("mousedown", handleDocumentMouseDown);
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }
    const highlightedOption = menuRef.current.querySelector(".dropdown-option.is-highlighted");
    if (highlightedOption !== null) {
      highlightedOption.scrollIntoView({ block: "nearest" });
    }
  }, [isOpen, highlightedIndex]);

  function open() {
    const selectedIndex = options.findIndex(option => option.value === value);
    setQuery("");
    setHighlightedIndex(Math.max(selectedIndex, 0));
    setIsOpen(true);
  }

  function select(option) {
    setIsOpen(false);
    if (option.value !== value) {
      onChange(option.value);
    }
  }

  function handleButtonClick() {
    if (isOpen) {
      setIsOpen(false);
    } else {
      open();
    }
  }

  function handleQueryChange(event) {
    setQuery(event.target.value);
    setHighlightedIndex(0);
  }

  // Escape stops here so it closes the menu without also closing the modal around it.
  function handleKeyDown(event) {
    if (!isOpen) {
      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        open();
      }
      return;
    }

    if (event.key === "ArrowDown") {
      event.preventDefault();
      setHighlightedIndex(previous => Math.min(previous + 1, filteredOptions.length - 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlightedIndex(previous => Math.max(previous - 1, 0));
    } else if (event.key === "Enter") {
      event.preventDefault();
      if (filteredOptions.length > 0) {
        select(filteredOptions[highlightedIndex]);
      }
    } else if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      setIsOpen(false);
    } else if (event.key === "Tab") {
      setIsOpen(false);
    }
  }

  const selectedOption = options.find(option => option.value === value);
  let buttonLabel = <span className="dropdown-placeholder">{placeholder}</span>;
  if (selectedOption !== undefined) {
    buttonLabel = <span className="dropdown-button-label">{selectedOption.label}</span>;
  }

  let labelElement = null;
  if (label !== undefined) {
    labelElement = <label htmlFor={id}>{label}</label>;
  }

  let hintElement = null;
  if (hint !== "") {
    hintElement = <div className="dropdown-hint">{hint}</div>;
  }

  let menu = null;
  if (isOpen) {
    let searchInput = null;
    if (isSearchable) {
      searchInput = <input className="dropdown-search" placeholder="Search..." value={query} onChange={handleQueryChange} onKeyDown={handleKeyDown} autoFocus />;
    }

    let optionElements = filteredOptions.map((option, index) => {
      const isSelected = option.value === value;

      let descriptionElement = null;
      if (option.description !== undefined) {
        descriptionElement = <span className="dropdown-option-description">{option.description}</span>;
      }

      let checkIcon = null;
      if (isSelected) {
        checkIcon = <CheckIcon />;
      }

      return (
        <li key={option.value} className={`dropdown-option ${index === highlightedIndex ? "is-highlighted" : ""}`} onMouseEnter={() => setHighlightedIndex(index)} onClick={() => select(option)}>
          <div className="dropdown-option-text">
            <span className="dropdown-option-label">{option.label}</span>
            {descriptionElement}
          </div>
          {checkIcon}
        </li>
      );
    });

    if (optionElements.length === 0) {
      optionElements = <li className="dropdown-empty">No matches</li>;
    }

    menu = (
      <div className="dropdown-menu">
        {searchInput}
        <ul ref={menuRef} className="dropdown-options">{optionElements}</ul>
      </div>
    );
  }

  return (
    <div ref={containerRef} className={`dropdown form-field-container ${isOpen ? "is-open" : ""}`}>
      {labelElement}
      {hintElement}
      <div className="dropdown-anchor">
        <button type="button" id={id} className="dropdown-button" onClick={handleButtonClick} onKeyDown={handleKeyDown}>
          {buttonLabel}
          <ChevronDownIcon />
        </button>
        {menu}
      </div>
    </div>
  );
}
