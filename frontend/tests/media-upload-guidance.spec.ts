import { expect, test } from '@playwright/test';
import {
	MEDIA_UPLOAD_LIMIT_BYTES,
	isOversizedMediaUpload,
	oversizedMediaUploadMessageKey
} from '../src/lib/media-upload-guidance';

test.describe('media upload guidance', () => {
	test('routes oversized videos to embed guidance', () => {
		const file = { name: 'demo.mov', size: MEDIA_UPLOAD_LIMIT_BYTES + 1, type: 'video/quicktime' };

		expect(isOversizedMediaUpload(file)).toBe(true);
		expect(oversizedMediaUploadMessageKey(file)).toBe('admin.media.toast_video_too_large');
	});

	test('detects videos by extension when MIME type is missing', () => {
		const file = { name: 'screen-recording.webm', size: MEDIA_UPLOAD_LIMIT_BYTES + 1, type: '' };

		expect(oversizedMediaUploadMessageKey(file)).toBe('admin.media.toast_video_too_large');
	});

	test('routes oversized non-video files to compression guidance', () => {
		const file = { name: 'portfolio.pdf', size: MEDIA_UPLOAD_LIMIT_BYTES + 1, type: 'application/pdf' };

		expect(oversizedMediaUploadMessageKey(file)).toBe('admin.media.toast_file_too_large_guidance');
	});
});
