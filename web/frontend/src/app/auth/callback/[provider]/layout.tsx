export function generateStaticParams() {
  return [
    { provider: 'google' },
    { provider: 'github' },
  ];
}

export default function CallbackLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
