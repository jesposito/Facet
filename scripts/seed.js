// Seed script for me.yaml
// Run this after first setup to create demo data
//
// Usage: node scripts/seed.js
//
// This creates sample profile data for demonstration purposes.

const PocketBase = require('pocketbase').default;

const pb = new PocketBase(process.env.POCKETBASE_URL || 'http://localhost:8090');

async function seed() {
  console.log('Seeding demo data...');

  try {
    // Create profile
    const profile = await pb.collection('profile').create({
      name: 'Alex Developer',
      headline: 'Full Stack Engineer building tools that matter',
      location: 'San Francisco, CA',
      summary: `I build software that helps people do their best work. Currently focused on developer tools and infrastructure.

With 10+ years of experience across startups and larger companies, I've shipped products used by millions of developers worldwide.

**What I do:**
- Design and build scalable backend systems
- Create intuitive developer experiences
- Lead and mentor engineering teams
- Bridge the gap between product and engineering`,
      contact_email: 'alex@example.com',
      contact_links: [
        { type: 'github', url: 'https://github.com/alexdev', label: 'GitHub' },
        { type: 'linkedin', url: 'https://linkedin.com/in/alexdev', label: 'LinkedIn' },
        { type: 'website', url: 'https://alexdev.io', label: 'Blog' }
      ],
      visibility: 'public'
    });
    console.log('✓ Created profile');

    // Create experiences
    const experiences = [
      {
        company: 'TechCorp',
        title: 'Staff Software Engineer',
        location: 'San Francisco, CA',
        start_date: '2022-01-01',
        end_date: null,
        description: 'Leading platform engineering initiatives for developer productivity.',
        bullets: [
          'Architected internal developer platform serving 500+ engineers',
          'Reduced CI/CD pipeline times by 60% through strategic caching',
          'Mentored 5 engineers to senior level promotions'
        ],
        skills: ['Go', 'Kubernetes', 'AWS', 'Terraform'],
        visibility: 'public',
        is_draft: false,
        sort_order: 1
      },
      {
        company: 'StartupXYZ',
        title: 'Senior Backend Engineer',
        location: 'Remote',
        start_date: '2019-03-01',
        end_date: '2021-12-31',
        description: 'Early engineer building the core platform from scratch.',
        bullets: [
          'Built real-time collaboration features handling 10k+ concurrent users',
          'Designed event-driven architecture for scalable data processing',
          'Established engineering best practices and code review culture'
        ],
        skills: ['Node.js', 'PostgreSQL', 'Redis', 'Docker'],
        visibility: 'public',
        is_draft: false,
        sort_order: 2
      }
    ];

    for (const exp of experiences) {
      await pb.collection('experience').create(exp);
    }
    console.log('✓ Created experiences');

    // Create projects
    const projects = [
      {
        title: 'DevFlow',
        slug: 'devflow',
        summary: 'An open-source developer workflow automation tool',
        description: `DevFlow helps teams automate repetitive development tasks.

## Features
- Git hooks automation
- CI/CD integration
- Custom workflow definitions
- Slack notifications`,
        tech_stack: ['Go', 'React', 'PostgreSQL', 'Docker'],
        links: [
          { type: 'github', url: 'https://github.com/example/devflow' },
          { type: 'website', url: 'https://devflow.dev' }
        ],
        categories: ['devtools', 'open-source', 'automation'],
        visibility: 'public',
        is_draft: false,
        is_featured: true,
        sort_order: 1
      },
      {
        title: 'CloudSync',
        slug: 'cloudsync',
        summary: 'Cross-cloud file synchronization service',
        description: 'A tool for keeping files synchronized across different cloud providers.',
        tech_stack: ['Rust', 'AWS', 'GCP', 'Azure'],
        links: [
          { type: 'github', url: 'https://github.com/example/cloudsync' }
        ],
        categories: ['cloud', 'infrastructure'],
        visibility: 'public',
        is_draft: false,
        is_featured: false,
        sort_order: 2
      }
    ];

    for (const project of projects) {
      await pb.collection('projects').create(project);
    }
    console.log('✓ Created projects');

    // Create education
    await pb.collection('education').create({
      institution: 'University of Technology',
      degree: 'Bachelor of Science',
      field: 'Computer Science',
      start_date: '2008-09-01',
      end_date: '2012-05-01',
      description: 'Focus on distributed systems and algorithms.',
      visibility: 'public',
      is_draft: false,
      sort_order: 1
    });
    console.log('✓ Created education');

    // Create skills
    const skillCategories = {
      'Languages': ['Go', 'TypeScript', 'Python', 'Rust', 'SQL'],
      'Frameworks': ['React', 'Node.js', 'FastAPI', 'Svelte'],
      'Infrastructure': ['Kubernetes', 'AWS', 'Docker', 'Terraform'],
      'Databases': ['PostgreSQL', 'Redis', 'MongoDB', 'SQLite']
    };

    let sortOrder = 1;
    for (const [category, skills] of Object.entries(skillCategories)) {
      for (const name of skills) {
        await pb.collection('skills').create({
          name,
          category,
          proficiency: sortOrder <= 3 ? 'expert' : sortOrder <= 6 ? 'proficient' : 'familiar',
          visibility: 'public',
          sort_order: sortOrder++
        });
      }
    }
    console.log('✓ Created skills');

    // Create a view
    await pb.collection('views').create({
      name: 'Technical Portfolio',
      slug: 'technical',
      description: 'Focused on technical projects and skills',
      visibility: 'public',
      hero_headline: 'Full Stack Engineer & Open Source Contributor',
      hero_summary: 'Building developer tools and infrastructure. Open to interesting technical challenges.',
      cta_text: 'View Resume',
      cta_url: '/resume.pdf',
      sections: JSON.stringify([
        { section: 'projects', enabled: true },
        { section: 'experience', enabled: true },
        { section: 'skills', enabled: true },
        { section: 'education', enabled: true }
      ]),
      is_active: true
    });
    console.log('✓ Created view');

    console.log('\n✅ Seed complete! Visit http://localhost:8080 to see your profile.');
  } catch (error) {
    console.error('Error seeding data:', error);
    process.exit(1);
  }
}

seed();
