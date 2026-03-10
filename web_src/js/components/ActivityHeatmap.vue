<script>
import {createTippy} from '../modules/tippy.js';

export default {
  props: {
    values: {
      type: Array,
      default: () => [],
    },
    locale: {
      type: Object,
      default: () => {},
    },
    firstDOW: {
      type: Number,
      default: 1, // Monday by default
    },
  },
  data() {
    return {
      endDate: new Date(),
    };
  },
  computed: {
    graphData() {
      const data = {};
      // Generate all days in the range (past 365 days)
      const startDate = new Date(Date.now() - 364 * 24 * 60 * 60 * 1000);
      const endDate = this.endDate;
      
      for (let d = new Date(startDate); d <= endDate; d.setDate(d.getDate() + 1)) {
        const dateStr = d.toISOString().split('T')[0];
        data[dateStr] = {
          parts: [],
        };
      }
      
      // Override with actual contribution data
      for (const {date, count} of this.values) {
        const dateStr = date.toISOString().split('T')[0];
        data[dateStr] = {
          parts: this.getLevel(count),
        };
      }
      return data;
    },
    // Map of date strings to contribution counts for quick lookup
    contributionMap() {
      const map = {};
      for (const {date, count} of this.values) {
        const dateStr = date.toISOString().split('T')[0];
        map[dateStr] = count;
      }
      return map;
    },
  },
  mounted() {
    this.$nextTick(() => {
      this.scrollToRecent();
      this.setupTooltips();
    });
  },
  beforeUnmount() {
    this.destroyTooltips();
  },
  methods: {
    scrollToRecent() {
      const wrapper = this.$el.querySelector('.activity-graph-scroll-wrapper');
      if (wrapper) {
        wrapper.scrollLeft = wrapper.scrollWidth;
      }
    },
    setupTooltips() {
      const graph = this.$el.querySelector('.forgejo-activity-graph');
      if (!graph) return;
      
      // Get all day elements from shadow DOM
      const shadow = graph.shadowRoot;
      if (!shadow) return;
      
      const days = shadow.querySelectorAll('[data-date]');
      days.forEach((day) => {
        if (day._tippy) return;
        
        const dateStr = day.dataset.date;
        const date = new Date(dateStr);
        const count = this.contributionMap[dateStr] || 0;

        createTippy(day, {
          content: this.formatTooltip(count, date),
          allowHTML: true,
          role: 'tooltip',
          theme: 'tooltip',
        });
      });
    },
    destroyTooltips() {
      const graph = this.$el.querySelector('.forgejo-activity-graph');
      if (!graph || !graph.shadowRoot) return;
      
      const days = graph.shadowRoot.querySelectorAll('[data-date]');
      days.forEach((day) => {
        if (day._tippy) {
          day._tippy.destroy();
        }
      });
    },
    getLevel(count) {
      if (count === 0) return [];
      const max = Math.max(...this.values.map(v => v.count), 1);
      const level = Math.min(Math.ceil((count / max) * 4), 4);
      return [`level-${level}`];
    },
    formatTooltip(count, date) {
      if (count === 0) {
        return this.locale.contributions_format
          .replace('{contributions}', `<b>${this.locale.contributions_zero}</b>`)
          .replace('{month}', this.locale.months[date.getMonth()])
          .replace('{day}', date.getDate())
          .replace('{year}', date.getFullYear());
      }
      const contributionsText = count === 1
        ? this.locale.contributions_one
        : this.locale.contributions_few;
      return this.locale.contributions_format
        .replace('{contributions}', `<b>${count} ${contributionsText}</b>`)
        .replace('{month}', this.locale.months[date.getMonth()])
        .replace('{day}', date.getDate())
        .replace('{year}', date.getFullYear());
    },
    handleDayClick(e) {
      const dateStr = e.composedPath()[0].dataset.date;
      if (!dateStr) return;

      const params = new URLSearchParams(document.location.search);
      const queryDate = params.get('date');

      if (queryDate && queryDate === dateStr) {
        params.delete('date');
      } else {
        params.set('date', dateStr);
      }

      params.delete('page');

      const newSearch = params.toString();
      window.location.search = newSearch.length ? `?${newSearch}` : '';
    },
  },
};
</script>
<template>
  <div class="activity-heatmap-wrapper">
    <div class="activity-graph-scroll-wrapper">
      <activity-graph
        :start-date="new Date(Date.now() - 364 * 24 * 60 * 60 * 1000)"
        :end-date="endDate"
        :data="graphData"
        :week-start-day="firstDOW"
        weekday-headers="narrow"
        month-headers="short"
        class="forgejo-activity-graph"
        @click="handleDayClick"
      />
    </div>
    <div class="heatmap-footer">
      <div class="total-contributions">
        {{ locale.contributions_in_the_last_12_months }}
      </div>
      <div class="heatmap-legend">
        <span class="heatmap-legend-label">{{ locale.less }}</span>
        <span class="heatmap-legend-box heatmap-legend-level-0"></span>
        <span class="heatmap-legend-box heatmap-legend-level-1"></span>
        <span class="heatmap-legend-box heatmap-legend-level-2"></span>
        <span class="heatmap-legend-box heatmap-legend-level-3"></span>
        <span class="heatmap-legend-box heatmap-legend-level-4"></span>
        <span class="heatmap-legend-label">{{ locale.more }}</span>
      </div>
    </div>
  </div>
</template>
