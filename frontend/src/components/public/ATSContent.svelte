<script lang="ts">
	import type { Profile, Experience, Education, Skill, ContactMethod, Certification } from '$lib/pocketbase';

	interface Props {
		profile: Profile | null;
		experience?: Experience[];
		education?: Education[];
		skills?: Skill[];
		contacts?: ContactMethod[];
		certifications?: Certification[];
	}

	let { profile, experience = [], education = [], skills = [], contacts = [], certifications = [] }: Props = $props();

	// Strip markdown/HTML to plain text for ATS parsing
	function stripToPlainText(text: string | undefined): string {
		if (!text) return '';
		// Remove markdown links [text](url) -> text
		let plain = text.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1');
		// Remove HTML tags
		plain = plain.replace(/<[^>]+>/g, '');
		// Remove markdown formatting
		plain = plain.replace(/[*_~`#]/g, '');
		// Normalize whitespace
		plain = plain.replace(/\s+/g, ' ').trim();
		return plain;
	}

	// Format date for ATS (YYYY-MM or "Present")
	function formatATSDate(date: string | undefined, isEndDate: boolean = false): string {
		if (!date) return isEndDate ? 'Present' : '';
		// Parse YYYY-MM-DD or YYYY-MM format
		const parts = date.split('-');
		if (parts.length >= 2) {
			const year = parts[0];
			const month = parseInt(parts[1], 10);
			const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
			return `${monthNames[month - 1]} ${year}`;
		}
		return date;
	}

	// Get important contacts for resume
	let emailContact = $derived(contacts.find(c => c.type === 'email'));
	let phoneContact = $derived(contacts.find(c => c.type === 'phone'));
	let linkedinContact = $derived(contacts.find(c => c.type === 'linkedin'));
	let githubContact = $derived(contacts.find(c => c.type === 'github'));
	let websiteContact = $derived(contacts.find(c => c.type === 'website' || c.type === 'portfolio'));

	// Check if we have any contact info at all
	let hasAnyContact = $derived(emailContact || phoneContact || linkedinContact || githubContact || websiteContact || contacts.length > 0);

	// Group skills by category for ATS
	let skillsByCategory = $derived(skills.reduce((acc, skill) => {
		const category = skill.category || 'Skills';
		if (!acc[category]) {
			acc[category] = [];
		}
		acc[category].push(skill.name);
		return acc;
	}, {} as Record<string, string[]>));

	// Flatten all skill names for keyword section
	let allSkillNames = $derived(skills.map(s => s.name));
</script>

<!--
	ATS Content - Hidden plain-text version for Applicant Tracking Systems
	This content is visually hidden but remains in the DOM for PDF text extraction.
	When users print to PDF, ATS systems can extract this structured text.
	Uses standard section headers that ATS systems recognize.
-->
<div id="ats-content" class="ats-only">
	<!-- CONTACT INFORMATION -->
	<section>
		<h2>CONTACT</h2>
		{#if profile?.name}
			<p>{profile.name}</p>
		{/if}
		{#if profile?.headline}
			<p>{profile.headline}</p>
		{/if}
		{#if profile?.location}
			<p>Location: {profile.location}</p>
		{/if}
		{#if emailContact}
			<p>Email: {emailContact.value}</p>
		{/if}
		{#if phoneContact}
			<p>Phone: {phoneContact.value}</p>
		{/if}
		{#if linkedinContact}
			<p>LinkedIn: {linkedinContact.value}</p>
		{/if}
		{#if githubContact}
			<p>GitHub: {githubContact.value}</p>
		{/if}
		{#if websiteContact}
			<p>Website: {websiteContact.value}</p>
		{/if}
		{#if !hasAnyContact && profile?.contact_email}
			<p>Email: {profile.contact_email}</p>
		{/if}
	</section>

	<!-- PROFESSIONAL SUMMARY -->
	{#if profile?.summary}
		<section>
			<h2>PROFESSIONAL SUMMARY</h2>
			<p>{stripToPlainText(profile.summary)}</p>
		</section>
	{/if}

	<!-- WORK EXPERIENCE -->
	{#if experience.length > 0}
		<section>
			<h2>EXPERIENCE</h2>
			{#each experience as job}
				<div>
					<p><strong>{job.title}</strong></p>
					<p>{job.company}{job.location ? ` | ${job.location}` : ''} | {formatATSDate(job.start_date)} - {formatATSDate(job.end_date, true)}</p>
					{#if job.description}
						<p>{stripToPlainText(job.description)}</p>
					{/if}
					{#if job.bullets && job.bullets.length > 0}
						{#each job.bullets as bullet}
							<p>* {stripToPlainText(bullet)}</p>
						{/each}
					{/if}
				</div>
			{/each}
		</section>
	{/if}

	<!-- EDUCATION -->
	{#if education.length > 0}
		<section>
			<h2>EDUCATION</h2>
			{#each education as edu}
				<div>
					<p><strong>{[edu.degree, edu.field].filter(Boolean).join(' in ')}</strong></p>
					<p>{edu.institution} | {formatATSDate(edu.start_date)} - {formatATSDate(edu.end_date, true)}</p>
					{#if edu.description}
						<p>{stripToPlainText(edu.description)}</p>
					{/if}
				</div>
			{/each}
		</section>
	{/if}

	<!-- SKILLS -->
	{#if skills.length > 0}
		<section>
			<h2>SKILLS</h2>
			{#each Object.entries(skillsByCategory) as [category, skillList]}
				<p>{category}: {skillList.join(', ')}</p>
			{/each}
		</section>
	{/if}

	<!-- CERTIFICATIONS -->
	{#if certifications.length > 0}
		<section>
			<h2>CERTIFICATIONS</h2>
			{#each certifications as cert}
				<p>{cert.name}{cert.issuer ? ` - ${cert.issuer}` : ''}{cert.issue_date ? ` (${formatATSDate(cert.issue_date)})` : ''}</p>
			{/each}
		</section>
	{/if}

	<!-- KEYWORDS (hidden skills list for ATS keyword matching) -->
	{#if allSkillNames.length > 0}
		<section>
			<h2>KEYWORDS</h2>
			<p>{allSkillNames.join(', ')}</p>
		</section>
	{/if}
</div>
