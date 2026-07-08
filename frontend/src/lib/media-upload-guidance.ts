export const MEDIA_UPLOAD_LIMIT_BYTES = 20 * 1024 * 1024;

type UploadCandidate = {
	name?: string;
	size: number;
	type?: string;
};

const VIDEO_EXTENSIONS = /\.(avi|m4v|mkv|mov|mp4|mpeg|mpg|ogv|webm|wmv)$/i;

export function isOversizedMediaUpload(file: UploadCandidate): boolean {
	return file.size > MEDIA_UPLOAD_LIMIT_BYTES;
}

export function isVideoUpload(file: UploadCandidate): boolean {
	if (file.type?.toLowerCase().startsWith('video/')) {
		return true;
	}
	return VIDEO_EXTENSIONS.test(file.name ?? '');
}

export function oversizedMediaUploadMessageKey(file: UploadCandidate): string {
	return isVideoUpload(file)
		? 'admin.media.toast_video_too_large'
		: 'admin.media.toast_file_too_large_guidance';
}
