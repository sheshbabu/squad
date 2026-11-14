import './Input.css';

export default function Input({ id, label = "", type = "text", placeholder, value, hint = "", error = "", isDisabled, onChange, rows = 4, list }) {
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
      <textarea id={id} name={id} placeholder={placeholder} className={error !== "" ? "has-error" : ""} disabled={isDisabled} value={value || ""} onChange={onChange} rows={rows} />
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
