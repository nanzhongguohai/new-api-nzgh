const CODEX_WINDOW_TOTALS = {
  fiveHour: 600,
  weekly: 800,
};

export const clampPercent = (value) => {
  const v = Number(value);
  if (!Number.isFinite(v)) return 0;
  return Math.max(0, Math.min(100, v));
};

export const normalizePlanType = (value) => {
  if (value == null) return '';
  return String(value).trim().toLowerCase();
};

const getWindowDurationSeconds = (windowData) => {
  const value = Number(windowData?.limit_window_seconds);
  if (!Number.isFinite(value) || value <= 0) return null;
  return value;
};

const classifyWindowByDuration = (windowData) => {
  const seconds = getWindowDurationSeconds(windowData);
  if (seconds == null) return null;
  return seconds >= 24 * 60 * 60 ? 'weekly' : 'fiveHour';
};

export const resolveRateLimitWindows = (data) => {
  const rateLimit = data?.rate_limit ?? {};
  const primary = rateLimit?.primary_window ?? null;
  const secondary = rateLimit?.secondary_window ?? null;
  const windows = [primary, secondary].filter(Boolean);
  const planType = normalizePlanType(data?.plan_type ?? rateLimit?.plan_type);

  let fiveHourWindow = null;
  let weeklyWindow = null;

  for (const windowData of windows) {
    const bucket = classifyWindowByDuration(windowData);
    if (bucket === 'fiveHour' && !fiveHourWindow) {
      fiveHourWindow = windowData;
      continue;
    }
    if (bucket === 'weekly' && !weeklyWindow) {
      weeklyWindow = windowData;
    }
  }

  if (planType === 'free') {
    if (!weeklyWindow) {
      weeklyWindow = primary ?? secondary ?? null;
    }
    return { fiveHourWindow: null, weeklyWindow };
  }

  if (!fiveHourWindow && !weeklyWindow) {
    return {
      fiveHourWindow: primary ?? null,
      weeklyWindow: secondary ?? null,
    };
  }

  if (!fiveHourWindow) {
    fiveHourWindow =
      windows.find((windowData) => windowData !== weeklyWindow) ?? null;
  }
  if (!weeklyWindow) {
    weeklyWindow =
      windows.find((windowData) => windowData !== fiveHourWindow) ?? null;
  }

  return { fiveHourWindow, weeklyWindow };
};

const buildWindowSummary = (windowType, windowData) => {
  if (!windowData) return null;
  const total = CODEX_WINDOW_TOTALS[windowType];
  if (!Number.isFinite(total) || total <= 0) return null;

  const usedPercent = clampPercent(windowData?.used_percent ?? 0);
  const used = Math.max(0, Math.min(total, Math.round((usedPercent / 100) * total)));
  const remaining = Math.max(0, total - used);

  return {
    window_type: windowType,
    total,
    used,
    remaining,
    used_percent: usedPercent,
    reset_at: windowData?.reset_at ?? null,
    reset_after_seconds: windowData?.reset_after_seconds ?? null,
    limit_window_seconds: windowData?.limit_window_seconds ?? null,
  };
};

export const buildCodexUsageSummary = (data) => {
  if (!data || typeof data !== 'object') return null;

  const { fiveHourWindow, weeklyWindow } = resolveRateLimitWindows(data);
  const fiveHourSummary = buildWindowSummary('fiveHour', fiveHourWindow);
  const weeklySummary = buildWindowSummary('weekly', weeklyWindow);
  const candidates = [fiveHourSummary, weeklySummary].filter(Boolean);

  if (candidates.length === 0) return null;

  const active = candidates.reduce((best, current) => {
    if (!best) return current;
    if (current.remaining < best.remaining) return current;
    if (current.remaining > best.remaining) return best;
    if (current.window_type === 'fiveHour' && best.window_type !== 'fiveHour') {
      return current;
    }
    return best;
  }, null);

  return {
    active_window_type: active.window_type,
    active_used: active.used,
    active_remaining: active.remaining,
    active_total: active.total,
    active_used_percent: active.used_percent,
    fetched_at: Math.floor(Date.now() / 1000),
    windows: {
      fiveHour: fiveHourSummary,
      weekly: weeklySummary,
    },
  };
};

export const extractCodexUsageSummary = (otherInfo) => {
  if (!otherInfo) return null;
  try {
    const parsed = typeof otherInfo === 'string' ? JSON.parse(otherInfo) : otherInfo;
    if (!parsed || typeof parsed !== 'object') return null;
    const summary = parsed.codex_usage_summary;
    return summary && typeof summary === 'object' ? summary : null;
  } catch (error) {
    return null;
  }
};

export const mergeCodexUsageSummary = (otherInfo, summary) => {
  let parsed = {};
  if (otherInfo) {
    try {
      parsed = typeof otherInfo === 'string' ? JSON.parse(otherInfo) : { ...otherInfo };
    } catch (error) {
      parsed = {};
    }
  }
  parsed.codex_usage_summary = summary;
  return JSON.stringify(parsed);
};

