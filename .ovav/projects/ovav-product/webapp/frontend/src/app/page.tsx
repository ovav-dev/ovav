import { Nav } from "@/components/Nav";
import { Hero } from "@/components/Hero";
import { Pricing } from "@/components/Pricing";
import { Profiles } from "@/components/Profiles";
import { Moat } from "@/components/Moat";
import { CTA } from "@/components/CTA";
import { Footer } from "@/components/Footer";

export default function Home() {
  return (
    <>
      <Nav />
      <main>
        <Hero />
        <Pricing />
        <Profiles />
        <Moat />
        <CTA />
      </main>
      <Footer />
    </>
  );
}
