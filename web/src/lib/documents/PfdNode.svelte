<script lang="ts">
	// Satu simpul diagram PFD (M4.10). Bentuknya mengikuti Flow Symbol baris:
	// process, alternate process, decision, connector, merge, document, atau
	// ng untuk disposisi NG di samping langkah.
	import { Handle, Position, type NodeProps } from '@xyflow/svelte';

	let { data }: NodeProps & { data: { label: string; symbol: string; symbolLabel: string } } = $props();
</script>

<Handle type="target" position={Position.Top} />
<div class="node" data-symbol={data.symbol} title={data.symbolLabel}>
	<span>{data.label || '·'}</span>
</div>
<Handle type="source" position={Position.Bottom} />
<Handle id="ng" type="source" position={Position.Right} />

<style>
	.node {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 180px;
		min-height: 48px;
		padding: 6px 10px;
		border: 1.5px solid var(--text-2);
		border-radius: 6px;
		background: var(--surface);
		color: var(--text);
		font-size: 13px;
		text-align: center;
		box-sizing: border-box;
	}

	.node[data-symbol='alternate_process'] {
		border-radius: 24px;
	}

	.node[data-symbol='decision'] {
		clip-path: polygon(50% 0, 100% 50%, 50% 100%, 0 50%);
		min-height: 64px;
		background: var(--warning-weak);
	}

	.node[data-symbol='connector'] {
		width: 56px;
		min-height: 56px;
		border-radius: 50%;
	}

	.node[data-symbol='merge'] {
		clip-path: polygon(0 0, 100% 0, 50% 100%);
		min-height: 56px;
		padding-bottom: 20px;
	}

	.node[data-symbol='document'] {
		border-radius: 6px 6px 40% 40% / 6px 6px 16px 16px;
	}

	.node[data-symbol='ng'] {
		width: 110px;
		min-height: 36px;
		border-style: dashed;
		border-color: var(--danger);
		background: var(--danger-weak);
		color: var(--danger);
	}
</style>
