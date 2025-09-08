# DisktroByte Assets Directory

This directory contains all the visual assets and images used in the project documentation.

## Directory Structure

```
assets/
├── images/
│   ├── logos/
│   │   ├── disktroByte-logo.png          # Main project logo
│   │   ├── disktroByte-logo-dark.png     # Dark theme logo
│   │   └── disktroByte-icon.ico          # Icon format
│   ├── screenshots/
│   │   ├── dashboard.png                 # Main dashboard screenshot
│   │   ├── dashboard-overview.png        # Dashboard overview
│   │   ├── file-reassembly.png           # File reassembly interface
│   │   ├── file-reassembly-interface.png # Detailed reassembly UI
│   │   ├── file-reassembly-banner.png    # Feature banner
│   │   ├── network-status.png            # Network monitoring view
│   │   ├── network-graph.png             # Network visualization
│   │   ├── network-topology.png          # Network topology diagram
│   │   ├── login-interface.png           # Authentication screen
│   │   ├── auth-flow.png                 # Authentication workflow
│   │   └── file-upload.png               # File upload interface
│   ├── demos/
│   │   ├── reassembly-demo.gif           # File reassembly animation
│   │   ├── progress-tracking.gif         # Progress tracking demo
│   │   ├── pdf-preview.png               # PDF file preview
│   │   ├── zip-preview.png               # ZIP file preview
│   │   ├── mp4-preview.png               # MP4 file preview
│   │   └── pptx-preview.png              # PPTX file preview
│   ├── diagrams/
│   │   ├── system-architecture.png       # System architecture diagram
│   │   ├── comparison-chart.png          # Feature comparison chart
│   │   ├── data-flow.png                 # Data flow diagram
│   │   └── security-model.png            # Security architecture
│   └── banners/
│       ├── hero-banner.png               # Main hero banner
│       ├── feature-banner.png            # Feature highlight banner
│       └── github-banner.png             # GitHub repository banner
└── docs/
    └── image-guidelines.md               # Image creation guidelines
```

## Image Specifications

### Screenshots
- **Format**: PNG
- **Resolution**: 1920x1080 (Full HD) recommended
- **Quality**: High quality, clear text visibility
- **Compression**: Optimized for web (keep file size reasonable)

### Logos
- **Format**: PNG with transparency
- **Sizes**: 
  - Main logo: 400x100px
  - Icon: 64x64px, 128x128px, 256x256px
- **Background**: Transparent
- **Colors**: Use brand colors consistently

### Demo GIFs
- **Format**: GIF
- **Duration**: 5-10 seconds maximum
- **Frame rate**: 10-15 FPS
- **Size**: Optimize for web (<2MB preferred)

### Diagrams
- **Format**: PNG or SVG
- **Style**: Clean, professional, consistent with project branding
- **Text**: Readable at various sizes
- **Colors**: Use project color scheme

## Usage in Documentation

All images are referenced in the README.md using relative paths:

```markdown
![Description](assets/images/category/filename.png)
```

## Creating Screenshots

### For GUI Screenshots:
1. Use consistent browser window size
2. Clean, default browser appearance
3. Show realistic data/content
4. Highlight key features with subtle overlays if needed

### For Demo GIFs:
1. Use screen recording tools (OBS, LICEcap, etc.)
2. Focus on specific features
3. Keep file sizes optimized
4. Show realistic user interactions

## Brand Guidelines

### Colors
- Primary: #2563eb (Blue)
- Secondary: #10b981 (Green)
- Accent: #f59e0b (Orange)
- Dark: #1f2937
- Light: #f8fafc

### Typography
- Headings: System fonts (San Francisco, Segoe UI, etc.)
- Code: Monospace fonts (Fira Code, Monaco, etc.)

## File Naming Convention

- Use lowercase with hyphens: `file-name.png`
- Be descriptive: `file-reassembly-progress-bar.png`
- Include version if needed: `dashboard-v2.png`
- Use consistent prefixes for categories

## Contributing Images

When adding new images:

1. Follow the directory structure
2. Use appropriate file formats
3. Optimize file sizes
4. Update this README if adding new categories
5. Test image links in documentation
6. Ensure all images are properly licensed

## Image Optimization Tools

Recommended tools for optimization:
- **TinyPNG**: For PNG compression
- **SVGO**: For SVG optimization
- **ImageOptim**: For general optimization
- **Figma/Sketch**: For creating diagrams and mockups

## Notes

- All images should be created or used with proper licensing
- Screenshots should not contain sensitive information
- Keep original high-resolution versions for future use
- Regular audit of unused images to keep repository clean
