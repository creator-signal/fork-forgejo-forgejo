import {mount} from '@vue/test-utils';
import ActivityHeatmap from './ActivityHeatmap.vue';
import {describe, expect, test, vi, beforeEach, afterEach} from 'vitest';

const mockLocale = {
  months: ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
  days: ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'],
  contributions_in_the_last_12_months: '100 contributions in the last 12 months',
  contributions_zero: 'No contributions',
  contributions_format: '{contributions} on {month} {day}, {year}',
  contributions_one: 'contribution',
  contributions_few: 'contributions',
  more: 'More',
  less: 'Less',
};

describe('ActivityHeatmap', () => {
  beforeEach(() => {
    // Mock createTippy to avoid DOM manipulation issues
    vi.mock('../modules/tippy.js', () => ({
      createTippy: vi.fn(),
    }));
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  test('renders with default props', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    expect(wrapper.find('.activity-heatmap-wrapper').exists()).toBe(true);
    expect(wrapper.find('.forgejo-activity-graph').exists()).toBe(true);
  });

  test('firstDOW prop defaults to 1 (Monday)', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    expect(wrapper.props('firstDOW')).toBe(1);
  });

  test('firstDOW prop accepts custom value', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
        firstDOW: 0, // Sunday
      },
    });

    expect(wrapper.props('firstDOW')).toBe(0);
  });

  test('contributionMap returns empty object for empty values', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    expect(wrapper.vm.contributionMap).toEqual({});
  });

  test('contributionMap correctly maps dates to counts', () => {
    const values = [
      {date: new Date('2024-01-15'), count: 5},
      {date: new Date('2024-01-16'), count: 10},
    ];

    const wrapper = mount(ActivityHeatmap, {
      props: {
        values,
        locale: mockLocale,
      },
    });

    const map = wrapper.vm.contributionMap;
    expect(map['2024-01-15']).toBe(5);
    expect(map['2024-01-16']).toBe(10);
  });

  test('getLevel returns empty array for zero count', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    expect(wrapper.vm.getLevel(0)).toEqual([]);
  });

  test('getLevel calculates correct level based on max value', () => {
    const values = [
      {date: new Date('2024-01-15'), count: 100},
    ];

    const wrapper = mount(ActivityHeatmap, {
      props: {
        values,
        locale: mockLocale,
      },
    });

    // With max=100: 25 -> level-1, 50 -> level-2, 75 -> level-3, 100 -> level-4
    expect(wrapper.vm.getLevel(25)).toEqual(['level-1']);
    expect(wrapper.vm.getLevel(50)).toEqual(['level-2']);
    expect(wrapper.vm.getLevel(75)).toEqual(['level-3']);
    expect(wrapper.vm.getLevel(100)).toEqual(['level-4']);
  });

  test('getLevel caps at level 4', () => {
    const values = [
      {date: new Date('2024-01-15'), count: 100},
    ];

    const wrapper = mount(ActivityHeatmap, {
      props: {
        values,
        locale: mockLocale,
      },
    });

    // Even with count > max, should cap at level-4
    expect(wrapper.vm.getLevel(150)).toEqual(['level-4']);
  });

  test('formatTooltip shows zero contributions message', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    const date = new Date('2024-03-15');
    const tooltip = wrapper.vm.formatTooltip(0, date);

    expect(tooltip).toContain('No contributions');
    expect(tooltip).toContain('March');
    expect(tooltip).toContain('15');
    expect(tooltip).toContain('2024');
  });

  test('formatTooltip shows single contribution message', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    const date = new Date('2024-03-15');
    const tooltip = wrapper.vm.formatTooltip(1, date);

    expect(tooltip).toContain('1 contribution');
    expect(tooltip).toContain('March');
    expect(tooltip).toContain('15');
    expect(tooltip).toContain('2024');
  });

  test('formatTooltip shows multiple contributions message', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    const date = new Date('2024-03-15');
    const tooltip = wrapper.vm.formatTooltip(5, date);

    expect(tooltip).toContain('5 contributions');
    expect(tooltip).toContain('March');
    expect(tooltip).toContain('15');
    expect(tooltip).toContain('2024');
  });

  test('graphData generates 365 days of data', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    const data = wrapper.vm.graphData;
    const dates = Object.keys(data);
    
    // Should have approximately 365 days (might be 364 or 365 depending on time math)
    expect(dates.length).toBeGreaterThanOrEqual(364);
    expect(dates.length).toBeLessThanOrEqual(366);
  });

  test('graphData merges contribution data correctly', () => {
    const today = new Date();
    const dateStr = today.toISOString().split('T')[0];
    
    const values = [
      {date: today, count: 10},
    ];

    const wrapper = mount(ActivityHeatmap, {
      props: {
        values,
        locale: mockLocale,
      },
    });

    const data = wrapper.vm.graphData;
    expect(data[dateStr]).toEqual({parts: ['level-4']});
  });

  test('activity-graph component receives correct weekStartDay prop', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
        firstDOW: 0, // Sunday
      },
    });

    const activityGraph = wrapper.find('activity-graph');
    expect(activityGraph.attributes('week-start-day')).toBe('0');
  });

  test('renders heatmap legend', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    expect(wrapper.find('.heatmap-legend').exists()).toBe(true);
    expect(wrapper.findAll('.heatmap-legend-box').length).toBe(5);
  });

  test('renders total contributions text', () => {
    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: mockLocale,
      },
    });

    expect(wrapper.find('.total-contributions').text()).toBe(mockLocale.contributions_in_the_last_12_months);
  });

  test('handles empty locale gracefully', () => {
    const emptyLocale = {
      months: [],
      days: [],
      contributions_in_the_last_12_months: '',
      contributions_zero: '',
      contributions_format: '{contributions} on {month} {day}, {year}',
      contributions_one: '',
      contributions_few: '',
      more: '',
      less: '',
    };

    const wrapper = mount(ActivityHeatmap, {
      props: {
        values: [],
        locale: emptyLocale,
      },
    });

    // Should not throw errors
    expect(wrapper.find('.activity-heatmap-wrapper').exists()).toBe(true);
  });
});
