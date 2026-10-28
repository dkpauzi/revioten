<script lang="ts">
	import HealthStatus from '#lib/HealthStatus.svelte';
	import { t } from '#lib/i18n/index.ts';
</script>

<h1>{t('homeTitle')}</h1>

<HealthStatus />
