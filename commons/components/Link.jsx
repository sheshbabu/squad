import navigateTo from '../utils/navigateTo.js';

export default function Link({ to, children, className = "", activeClassName = "", isActive, onClick }) {
  function handleClick(event) {
    if (event.metaKey || event.ctrlKey) {
      return;
    }
    event.stopPropagation();
    event.preventDefault();
    navigateTo(to);
    if (onClick !== undefined) {
      onClick();
    }
  }

  let finalClassName = className;
  if (isActive) {
    finalClassName += " " + activeClassName;
  }

  return (
    <a href={to} onClick={handleClick} className={finalClassName}>
      {children}
    </a>
  );
}
