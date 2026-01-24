import { dev } from '$app/environment';

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

const shouldLog = (level: LogLevel): boolean => {
	if (level === 'error') return true;
	if (level === 'warn') return true;
	return dev;
};

export const logger = {
	debug: (...args: unknown[]) => {
		if (shouldLog('debug')) console.log(...args);
	},
	info: (...args: unknown[]) => {
		if (shouldLog('info')) console.info(...args);
	},
	warn: (...args: unknown[]) => {
		if (shouldLog('warn')) console.warn(...args);
	},
	error: (...args: unknown[]) => {
		if (shouldLog('error')) console.error(...args);
	}
};

export default logger;
