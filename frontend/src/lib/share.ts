export interface ShareData {
	url: string;
	title: string;
	text?: string;
}

export interface ShareResult {
	method: 'native' | 'clipboard' | 'external';
	success: boolean;
	cancelled?: boolean;
}

export function canUseNativeShare(): boolean {
	return (
		typeof navigator !== 'undefined' &&
		'share' in navigator &&
		typeof window !== 'undefined' &&
		window.isSecureContext
	);
}

export async function nativeShare(data: ShareData): Promise<ShareResult> {
	if (!canUseNativeShare()) {
		return { method: 'native', success: false };
	}

	try {
		await navigator.share({
			url: data.url,
			title: data.title,
			text: data.text
		});
		return { method: 'native', success: true };
	} catch (error: unknown) {
		if (error instanceof Error && error.name === 'AbortError') {
			return { method: 'native', success: false, cancelled: true };
		}
		throw error;
	}
}

export async function copyToClipboard(text: string): Promise<boolean> {
	if (navigator.clipboard) {
		try {
			await navigator.clipboard.writeText(text);
			return true;
		} catch {}
	}

	const textarea = document.createElement('textarea');
	textarea.value = text;
	textarea.style.position = 'fixed';
	textarea.style.left = '-999999px';
	textarea.setAttribute('readonly', '');
	document.body.appendChild(textarea);
	textarea.select();
	try {
		document.execCommand('copy');
		return true;
	} catch {
		return false;
	} finally {
		document.body.removeChild(textarea);
	}
}

export interface ShareUrls {
	linkedin: string;
	twitter: string;
	reddit: string;
	email: string;
}

export function getShareUrls(data: ShareData): ShareUrls {
	const encodedUrl = encodeURIComponent(data.url);
	const encodedTitle = encodeURIComponent(data.title);
	const shareText = data.text || data.title;
	const encodedText = encodeURIComponent(shareText);
	const emailBody = encodeURIComponent((data.text || data.title) + '\n\n' + data.url);

	return {
		linkedin: `https://www.linkedin.com/sharing/share-offsite/?url=${encodedUrl}`,
		twitter: `https://twitter.com/intent/tweet?text=${encodedText}&url=${encodedUrl}`,
		reddit: `https://www.reddit.com/submit?url=${encodedUrl}&title=${encodedTitle}`,
		email: `mailto:?subject=${encodedTitle}&body=${emailBody}`
	};
}
