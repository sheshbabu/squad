import './Input.css';

const { useRef, useLayoutEffect } = React;

export default function Input({ id, label = "", type = "text", placeholder, value, hint = "", error = "", isDisabled, onChange, rows = 4, list }) {
  const textareaRef = useRef(null);

  // Safari and Firefox don't support field-sizing, so the textarea grows to fit its content here instead.
  useLayoutEffect(() => {
    const textarea = textareaRef.current;
    if (textarea === null) {
      return;
    }
    textarea.style.height = "auto";
    textarea.style.height = `${textarea.scrollHeight + textarea.offsetHeight - textarea.clientHeight}px`;
  }, [value]);

  let hintElement = null;
  if (hint !== "") {
    hintElement = <div className="hint">{hint}</div>;
  }

  let errorElement = null;
  if (error !== "") {
    errorElement = <div className="error">{error}</div>;
  }

  let labelElement = null;
  if (label !== "") {
    labelElement = <label htmlFor={id}>{label}</label>;
  }

  let inputElement = (
    <input type={type} id={id} name={id} placeholder={placeholder} className={error !== "" ? "has-error" : ""} disabled={isDisabled} value={value || ""} onChange={onChange} list={list} />
  );
  if (type === "textarea") {
    inputElement = (
      <textarea ref={textareaRef} id={id} name={id} placeholder={placeholder} className={error !== "" ? "has-error" : ""} disabled={isDisabled} value={value || ""} onChange={onChange} rows={rows} />
    );
  }

  return (
    <div className="input-container form-field-container">
      {labelElement}
      {hintElement}
      {inputElement}
      {errorElement}
    </div>
  );
}
