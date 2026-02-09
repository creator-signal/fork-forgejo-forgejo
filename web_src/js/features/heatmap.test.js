import {expect, test} from 'vitest';
import {buildWeekdayLabels, normalizeWeekStart, shiftDate, weekStartToIndex} from './heatmap.js';

test('normalizeWeekStart', () => {
  expect(normalizeWeekStart('MONDAY')).toEqual('monday');
  expect(normalizeWeekStart('monday')).toEqual('monday');
  expect(normalizeWeekStart('sunday')).toEqual('sunday');
  expect(normalizeWeekStart('tuesday')).toEqual('tuesday');
  expect(normalizeWeekStart('wednesday')).toEqual('wednesday');
  expect(normalizeWeekStart('thursday')).toEqual('thursday');
  expect(normalizeWeekStart('friday')).toEqual('friday');
  expect(normalizeWeekStart('saturday')).toEqual('saturday');
  expect(normalizeWeekStart('')).toEqual('monday');
  expect(normalizeWeekStart(null)).toEqual('monday');
  expect(normalizeWeekStart('noday')).toEqual('monday');
});

test('buildWeekdayLabels', () => {
  const originalLang = document.documentElement.lang;
  document.documentElement.lang = 'en-US';

  const sundayFirst = buildWeekdayLabels(weekStartToIndex.sunday);
  expect(sundayFirst[0]).toEqual('Sun');
  expect(sundayFirst[6]).toEqual('Sat');

  const mondayFirst = buildWeekdayLabels(weekStartToIndex.monday);
  expect(mondayFirst[0]).toEqual('Mon');
  expect(mondayFirst[6]).toEqual('Sun');

  const wednesdayFirst = buildWeekdayLabels(weekStartToIndex.wednesday);
  expect(wednesdayFirst[0]).toEqual('Wed');
  expect(wednesdayFirst[6]).toEqual('Tue');

  document.documentElement.lang = originalLang;
});

test('shiftDate', () => {
  const base = new Date(Date.UTC(2025, 0, 2));
  const shifted = shiftDate(base, -1);
  expect(shifted.getUTCFullYear()).toEqual(2025);
  expect(shifted.getUTCMonth()).toEqual(0);
  expect(shifted.getUTCDate()).toEqual(1);
});
