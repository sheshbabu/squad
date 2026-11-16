import Input from "../../commons/components/Input.jsx";
import Button from "../../commons/components/Button.jsx";
import { ArrowRightIcon } from "../../commons/components/Icon.jsx";
import ApiClient from "../../commons/http/ApiClient.js";
import "./LoginPage.css";

const { useState } = React;

export default function LoginPage({ isOnboarding = false }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailError, setEmailError] = useState("");
  const [passwordError, setPasswordError] = useState("");

  function handleSubmit(event) {
    event.preventDefault();

    setEmailError("");
    setPasswordError("");

    let promise;
    if (isOnboarding) {
      promise = ApiClient.createUser({ email, password });
    } else {
      promise = ApiClient.login({ email, password });
    }

    promise
      .then(() => {
        window.location.href = "/";
      })
      .catch(error => {
        switch (error.code) {
          case "INVALID_EMAIL":
            setEmailError("Invalid email address");
            break;
          case "INVALID_PASSWORD":
            setPasswordError("Invalid password");
            break;
          case "INCORRECT_EMAIL":
            setEmailError("Incorrect email address");
            break;
          case "INCORRECT_PASSWORD":
            setPasswordError("Incorrect password");
            break;
        }
      });
  }

  let header = <div className="login-page-title">Login</div>;
  let buttonText = "Login";

  if (isOnboarding) {
    header = (
      <div>
        <div className="login-page-title">Let's get started!</div>
        <div className="login-page-subtitle">Create your account.</div>
      </div>
    );
    buttonText = "Continue";
  }

  return (
    <form className="login-page" onSubmit={handleSubmit}>
      {header}
      <Input id="email" label="Email" type="email" placeholder="Enter your email address" value={email} error={emailError} onChange={event => setEmail(event.target.value)} />
      <Input id="password" label="Password" type="password" placeholder="Enter your password" value={password} error={passwordError} onChange={event => setPassword(event.target.value)} />
      <Button variant="primary" type="submit">{buttonText}<ArrowRightIcon /></Button>
    </form>
  );
}
