const deviceStorageKey = "device_id";

export function getDeviceId(): string {
  const existing = localStorage.getItem(deviceStorageKey);
  if (existing) {
    return existing;
  }

  const deviceId = crypto.randomUUID();
  localStorage.setItem(deviceStorageKey, deviceId);
  return deviceId;
}

export function phoneModel(): string {
  const uaData = navigator.userAgentData;
  if (uaData?.platform) {
    return uaData.platform;
  }

  return navigator.platform || "unknown";
}

export function browserName(): string {
  const ua = navigator.userAgent;
  if (ua.includes("Edg/")) return "Microsoft Edge";
  if (ua.includes("Chrome/")) return "Chrome";
  if (ua.includes("Safari/") && !ua.includes("Chrome/")) return "Safari";
  if (ua.includes("Firefox/")) return "Firefox";

  return "unknown";
}

declare global {
  interface Navigator {
    userAgentData?: {
      platform?: string;
    };
  }
}
