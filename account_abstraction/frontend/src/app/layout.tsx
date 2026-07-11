import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "TigerWallet - Account Abstraction",
  description: "Smart Accounts with ERC-4337 - Gasless transactions, social recovery, and more",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased min-h-screen bg-gradient-to-br from-tiger-dark to-black">
        {children}
      </body>
    </html>
  );
}
