// The web build gets no notifications (see notifications.ts).
export const hasBackgroundNotificationSwitch = false;

export async function backgroundNotificationsEnabled(): Promise<boolean> {
  return false;
}

export async function setBackgroundNotifications(_on: boolean): Promise<boolean> {
  return false;
}

export async function startNotifications(): Promise<void> {}

export async function stopNotifications(): Promise<void> {}
