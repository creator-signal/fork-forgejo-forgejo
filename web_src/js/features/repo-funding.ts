import {showModal} from "../modules/modal.ts";

export function initDonationButton() {
  const donationButton = document.querySelector('#donation-button');
  donationButton?.addEventListener('click', () => {
    showModal('funding-modal', () => {});
  });
}
