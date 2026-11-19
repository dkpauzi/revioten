// Bantuan layar dasbor admin (D-205).
import { t } from '../i18n/index.ts';

// waited menulis lama menunggu: "5 h" di bawah sehari, "2 d 3 h" sesudahnya.
export function waited(hours: number): string {
	const whole = Math.max(0, Math.floor(hours));
	if (whole < 24) return t('adminHours', { hours: String(whole) });
	return t('adminDays', { days: String(Math.floor(whole / 24)), hours: String(whole % 24) });
}
