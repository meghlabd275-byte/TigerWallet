import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "TigerWallet Multi-Chain Registry",
  description: "Multi-chain blockchain registry",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
