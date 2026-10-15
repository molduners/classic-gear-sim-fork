import { checkBagExport, findQuickSimTarget, startQuickSim } from '../core/quick_sim';

const form = document.querySelector<HTMLFormElement>('.quick-sim-form');
const input = document.querySelector<HTMLTextAreaElement>('.quick-sim-input');
const bagsInput = document.querySelector<HTMLTextAreaElement>('.quick-sim-bags-input');
const status = document.querySelector<HTMLElement>('.quick-sim-status');

form?.addEventListener('submit', event => {
	event.preventDefault();
	if (!input || !status) return;
	status.classList.remove('text-danger');
	if (!input.value.trim()) {
		status.textContent = 'Paste the text from the exporter addon first.';
		status.classList.add('text-danger');
		return;
	}
	try {
		const target = findQuickSimTarget(input.value);
		const bags = bagsInput?.value ?? '';
		if (bags.trim()) {
			checkBagExport(bags);
		}
		status.textContent = `Opening the ${target.label} sim…` + (target.validated ? '' : " (this spec's numbers haven't been checked yet)");
		window.location.href = startQuickSim(input.value, target, import.meta.env.BASE_URL, bags);
	} catch (e) {
		status.textContent = e instanceof Error ? e.message : String(e);
		status.classList.add('text-danger');
	}
});
