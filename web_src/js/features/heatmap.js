import {createApp} from 'vue';
import {translateMonth, translateDay} from '../utils.js';

export function shiftDate(date, days) {
  const shifted = new Date(date);
  shifted.setDate(shifted.getDate() + days);
  return shifted;
}

export const weekStartToIndex = {
  sunday: 0,
  monday: 1,
  tuesday: 2,
  wednesday: 3,
  thursday: 4,
  friday: 5,
  saturday: 6,
};

export function normalizeWeekStart(value) {
  const normalized = (value || '').toLowerCase();
  return Object.hasOwn(weekStartToIndex, normalized) ? normalized : 'monday';
}

export function buildWeekdayLabels(offset) {
  const days = new Array(7).fill().map((_, idx) => translateDay(idx));
  const normalizedOffset = ((offset % 7) + 7) % 7;
  if (normalizedOffset > 0) {
    days.push(...days.splice(0, normalizedOffset));
  }
  return days;
}

export async function initHeatmap() {
  const el = document.getElementById('user-heatmap');
  if (!el) return;

  const {default: ActivityHeatmap} = await import(/* webpackChunkName: "activity-heatmap" */'../components/ActivityHeatmap.vue');
  try {
    const requestedWeekStart = el.getAttribute('data-week-start') || 'monday';
    const weekStart = normalizeWeekStart(requestedWeekStart);
    const weekStartOffset = weekStartToIndex[weekStart];
    const displayShiftDays = -weekStartOffset;

    const heatmap = {};
    for (const {contributions, timestamp} of JSON.parse(el.getAttribute('data-heatmap-data'))) {
      // Convert to user timezone and sum contributions by date
      const dateStr = shiftDate(new Date(timestamp * 1000), displayShiftDays).toDateString();
      heatmap[dateStr] = (heatmap[dateStr] || 0) + contributions;
    }

    const values = Object.keys(heatmap).map((v) => {
      return {date: new Date(v), count: heatmap[v]};
    });

    const locale = {
      months: new Array(12).fill().map((_, idx) => translateMonth(idx)),
      days: buildWeekdayLabels(weekStartOffset),
      contributions_in_the_last_12_months: el.getAttribute('data-locale-total-contributions'),
      contributions_zero: el.getAttribute('data-locale-contributions-zero'),
      contributions_format: el.getAttribute('data-locale-contributions-format'),
      contributions_one: el.getAttribute('data-locale-contributions-one'),
      contributions_few: el.getAttribute('data-locale-contributions-few'),
      more: el.getAttribute('data-locale-more'),
      less: el.getAttribute('data-locale-less'),
    };

    const View = createApp(ActivityHeatmap, {values, locale, weekStartOffset});
    View.mount(el);
    el.classList.remove('is-loading');
  } catch (err) {
    console.error('Heatmap failed to load', err);
    el.textContent = 'Heatmap failed to load';
  }
}
