import ApiClient from "../http/ApiClient.js";

const { useEffect, useState } = React;

export default function useAuth() {
  const [isLoading, setIsLoading] = useState(true);
  const [currentUser, setCurrentUser] = useState(null);
  const [shouldShowLogin, setShouldShowLogin] = useState(false);
  const [shouldShowOnboarding, setShouldShowOnboarding] = useState(false);

  useEffect(() => {
    ApiClient.getCurrentUser()
      .then(setCurrentUser)
      .catch(error => {
        if (error.code === "NO_USERS") {
          setShouldShowOnboarding(true);
        } else if (error.code === "NO_SESSION") {
          setShouldShowLogin(true);
        }
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, []);

  return { isLoading, currentUser, shouldShowLogin, shouldShowOnboarding };
}
