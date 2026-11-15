import { showToast } from "../components/Toast.jsx";

async function request(method, url, payload) {
  const options = {
    method: method,
    headers: {},
  };

  if (payload instanceof FormData) {
    options.body = payload;
  } else if (payload !== undefined) {
    options.headers['Content-Type'] = 'application/json';
    options.body = JSON.stringify(payload);
  }

  try {
    const response = await fetch(url, options);

    if (!response.ok) {
      throw response;
    }

    const isJsonResponse = response.headers.get('content-type')?.includes('application/json');
    return isJsonResponse ? await response.json() : null;
  } catch (error) {
    if (!navigator.onLine) {
      showToast("No internet connection.");
      throw error;
    }

    if (error instanceof TypeError) {
      showToast("Connection failed.");
      throw error;
    }

    if (error instanceof Response) {
      const isJsonResponse = error.headers.get('content-type')?.includes('application/json');
      let body = null;
      if (isJsonResponse) {
        body = await error.json();
      }

      const err = new Error(body?.message || error.statusText);
      err.code = body?.code;
      err.status = error.status;

      // Login, onboarding and password change show these inline, and a lost session sends the person back to login.
      const skipCodes = ['NO_USERS', 'NO_SESSION', 'INVALID_NAME', 'INVALID_EMAIL', 'INVALID_PASSWORD', 'INCORRECT_EMAIL', 'INCORRECT_PASSWORD', 'INCORRECT_OLD_PASSWORD', 'INVALID_NEW_PASSWORD'];
      if (err.code === 'NO_SESSION' && url !== '/api/users/me/') {
        window.location.reload();
      } else if (!skipCodes.includes(err.code)) {
        showToast(err.message || 'An unexpected error occurred');
      }

      throw err;
    }

    throw error;
  }
}

// Users

function getCurrentUser() {
  return request('GET', '/api/users/me/');
}

function createUser(payload) {
  return request('POST', '/api/users/', payload);
}

function login(payload) {
  return request('POST', '/api/users/login/', payload);
}

function logout() {
  return request('POST', '/api/users/logout/');
}

function updatePassword(payload) {
  return request('PUT', '/api/users/me/password/', payload);
}

// Gateways

function getGateways() {
  return request('GET', '/api/gateways/');
}

function getModels(gateway) {
  return request('GET', `/api/gateways/${gateway}/models/`);
}

function getProviders(gateway, modelId) {
  return request('GET', `/api/gateways/${gateway}/providers/?model=${encodeURIComponent(modelId)}`);
}

function getZDREndpoints(gateway) {
  return request('GET', `/api/gateways/${gateway}/zdr/`);
}

// Bots

function getBots() {
  return request('GET', '/api/bots/');
}

function createBot(payload) {
  return request('POST', '/api/bots/', payload);
}

function updateBot(botId, payload) {
  return request('PUT', `/api/bots/${botId}/`, payload);
}

function deleteBot(botId) {
  return request('DELETE', `/api/bots/${botId}/`);
}

function updateBotTools(botId, payload) {
  return request('PUT', `/api/bots/${botId}/tools/`, payload);
}

function regenerateBotWebhook(botId) {
  return request('PUT', `/api/bots/${botId}/webhook/`);
}

function deleteBotWebhook(botId) {
  return request('DELETE', `/api/bots/${botId}/webhook/`);
}

function getToolGroups() {
  return request('GET', '/api/tools/');
}

// Memories

function getMemories(botId) {
  return request('GET', `/api/bots/${botId}/memories/`);
}

function deleteMemory(botId, memoryId) {
  return request('DELETE', `/api/bots/${botId}/memories/${memoryId}/`);
}

// Files

function getFiles(botId) {
  return request('GET', `/api/bots/${botId}/files/`);
}

function getFile(botId, name) {
  return request('GET', `/api/bots/${botId}/files/${encodeURIComponent(name)}/`);
}

function saveFile(botId, name, content) {
  return request('PUT', `/api/bots/${botId}/files/${encodeURIComponent(name)}/`, { content });
}

function deleteFile(botId, name) {
  return request('DELETE', `/api/bots/${botId}/files/${encodeURIComponent(name)}/`);
}

// Schedules

function getSchedules(botId) {
  return request('GET', `/api/bots/${botId}/schedules/`);
}

function createSchedule(botId, payload) {
  return request('POST', `/api/bots/${botId}/schedules/`, payload);
}

function updateSchedule(botId, scheduleId, payload) {
  return request('PUT', `/api/bots/${botId}/schedules/${scheduleId}/`, payload);
}

function deleteSchedule(botId, scheduleId) {
  return request('DELETE', `/api/bots/${botId}/schedules/${scheduleId}/`);
}

// Threads

function getThreads(botId) {
  return request('GET', `/api/bots/${botId}/threads/`);
}

function createThread(botId, payload) {
  return request('POST', `/api/bots/${botId}/threads/`, payload);
}

function getThread(threadId) {
  return request('GET', `/api/threads/${threadId}/`);
}

function getEarlierMessages(threadId, beforeMessageId) {
  return request('GET', `/api/threads/${threadId}/messages/?beforeMessageId=${beforeMessageId}`);
}

function sendMessage(threadId, payload) {
  return request('POST', `/api/threads/${threadId}/messages/`, payload);
}

function retryThread(threadId) {
  return request('POST', `/api/threads/${threadId}/retry/`);
}

function cancelThread(threadId) {
  return request('POST', `/api/threads/${threadId}/cancel/`);
}

function renameThread(threadId, title) {
  return request('PUT', `/api/threads/${threadId}/`, { title });
}

function getThreadUsage(threadId) {
  return request('GET', `/api/threads/${threadId}/usage/`);
}

function deleteThread(threadId) {
  return request('DELETE', `/api/threads/${threadId}/`);
}

function uploadAttachment(botId, file) {
  const formData = new FormData();
  formData.append('file', file);
  return request('POST', `/api/bots/${botId}/attachments/`, formData);
}

// Search

function search(query) {
  return request('GET', `/api/search/?query=${encodeURIComponent(query)}`);
}

export default {
  getCurrentUser,
  createUser,
  login,
  logout,
  updatePassword,
  getGateways,
  getModels,
  getProviders,
  getZDREndpoints,
  getBots,
  createBot,
  updateBot,
  deleteBot,
  updateBotTools,
  regenerateBotWebhook,
  deleteBotWebhook,
  getToolGroups,
  getMemories,
  deleteMemory,
  getFiles,
  getFile,
  saveFile,
  deleteFile,
  getSchedules,
  createSchedule,
  updateSchedule,
  deleteSchedule,
  getThreads,
  createThread,
  getThread,
  getEarlierMessages,
  sendMessage,
  retryThread,
  cancelThread,
  renameThread,
  getThreadUsage,
  deleteThread,
  uploadAttachment,
  search,
};
