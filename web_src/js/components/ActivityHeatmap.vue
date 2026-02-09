<script>
import {CalendarHeatmap} from 'vue3-calendar-heatmap';

export default {
  components: {CalendarHeatmap},
  props: {
    values: {
      type: Array,
      default: () => [],
    },
    locale: {
      type: Object,
      default: () => {},
    },
    weekStartOffset: {
      type: Number,
      default: 0,
    },
  },
  data() {
    const endDate = new Date();
    endDate.setDate(endDate.getDate() - this.weekStartOffset);
    return {
      colorRange: [
        'var(--color-secondary-alpha-60)',
        'var(--color-secondary-alpha-60)',
        'var(--color-primary-light-4)',
        'var(--color-primary-light-2)',
        'var(--color-primary)',
        'var(--color-primary-dark-2)',
        'var(--color-primary-dark-4)',
      ],
      endDate,
    };
  },
  mounted() {
    // work around issue with first legend color being rendered twice and legend cut off
    const legend = document.querySelector('.vch__external-legend-wrapper');
    legend.setAttribute('viewBox', '12 0 80 10');
    legend.style.marginRight = '-12px';
  },
  methods: {
    normalizeHeatmapDate(date) {
      if (!this.weekStartOffset) return date;
      const normalized = new Date(date);
      normalized.setDate(normalized.getDate() + this.weekStartOffset);
      return normalized;
    },
    handleDayClick(e) {
      // Reset filter if same date is clicked
      const params = new URLSearchParams(document.location.search);
      const queryDate = params.get('date');
      // Timezone has to be stripped because toISOString() converts to UTC
      const displayDate = this.normalizeHeatmapDate(e.date);
      const clickedDate = new Date(displayDate - (displayDate.getTimezoneOffset() * 60000)).toISOString().substring(0, 10);

      if (queryDate && queryDate === clickedDate) {
        params.delete('date');
      } else {
        params.set('date', clickedDate);
      }

      params.delete('page');

      const newSearch = params.toString();
      window.location.search = newSearch.length ? `?${newSearch}` : '';
    },
  },
};
</script>
<template>
  <div class="total-contributions">
    {{ locale.contributions_in_the_last_12_months }}
  </div>
  <calendar-heatmap
    :locale="locale"
    :no-data-text="locale.contributions_zero"
    :tooltip-formatter="
      (v) => {
        const displayDate = normalizeHeatmapDate(v.date);
        return locale.contributions_format
          .replace(
            '{contributions}',
            `<b>${v.count} ${
              v.count === 1
                ? locale.contributions_one
                : locale.contributions_few
            }</b>`
          )
          .replace('{month}', locale.months[displayDate.getMonth()])
          .replace('{day}', displayDate.getDate())
          .replace('{year}', displayDate.getFullYear());
      }
    "
    :end-date="endDate"
    :values="values"
    :range-color="colorRange"
    @day-click="handleDayClick($event)"
  />
</template>
